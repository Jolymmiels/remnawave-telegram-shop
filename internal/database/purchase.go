package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceType string

const (
	InvoiceTypeCrypto           InvoiceType = "crypto"
	InvoiceTypeYookasa          InvoiceType = "yookasa"
	InvoiceTypeTelegram         InvoiceType = "telegram"
	InvoiceTypeTribute          InvoiceType = "tribute"
	InvoiceTypePlatega          InvoiceType = "platega"
	InvoiceTypePlategaSBP       InvoiceType = "plt_sbp" // per-method types predate the single Platega link; kept for history
	InvoiceTypePlategaCards     InvoiceType = "plt_cards"
	InvoiceTypePlategaAcquiring InvoiceType = "plt_acq"
	InvoiceTypePlategaWorldwide InvoiceType = "plt_ww"
	InvoiceTypePlategaCrypto    InvoiceType = "plt_crypto"
	InvoiceTypeWata             InvoiceType = "wata"
)

type PurchaseStatus string

const (
	PurchaseStatusNew     PurchaseStatus = "new"
	PurchaseStatusPending PurchaseStatus = "pending"
	PurchaseStatusPaid    PurchaseStatus = "paid"
	PurchaseStatusCancel  PurchaseStatus = "cancel"
)

type Purchase struct {
	ID                int64          `db:"id"`
	Amount            float64        `db:"amount"`
	CustomerID        int64          `db:"customer_id"`
	CreatedAt         time.Time      `db:"created_at"`
	Month             int            `db:"month"`
	PaidAt            *time.Time     `db:"paid_at"`
	Currency          string         `db:"currency"`
	ExpireAt          *time.Time     `db:"expire_at"`
	Status            PurchaseStatus `db:"status"`
	InvoiceType       InvoiceType    `db:"invoice_type"`
	CryptoInvoiceID   *int64         `db:"crypto_invoice_id"`
	CryptoInvoiceLink *string        `db:"crypto_invoice_url"`
	YookasaURL        *string        `db:"yookasa_url"`
	YookasaID         *uuid.UUID     `db:"yookasa_id"`
	PlategaID         *string        `db:"platega_id"`
	PlategaURL        *string        `db:"platega_url"`
	WataID            *string        `db:"wata_id"`
	WataURL           *string        `db:"wata_url"`
}

type PurchaseRepository struct {
	pool *pgxpool.Pool
}

func NewPurchaseRepository(pool *pgxpool.Pool) *PurchaseRepository {
	return &PurchaseRepository{
		pool: pool,
	}
}

func (cr *PurchaseRepository) Create(ctx context.Context, purchase *Purchase) (int64, error) {
	buildInsert := sq.Insert("purchase").
		Columns("amount", "customer_id", "month", "currency", "expire_at", "status", "invoice_type", "crypto_invoice_id", "crypto_invoice_url", "yookasa_url", "yookasa_id", "platega_id", "platega_url", "wata_id", "wata_url").
		Values(purchase.Amount, purchase.CustomerID, purchase.Month, purchase.Currency, purchase.ExpireAt, purchase.Status, purchase.InvoiceType, purchase.CryptoInvoiceID, purchase.CryptoInvoiceLink, purchase.YookasaURL, purchase.YookasaID, purchase.PlategaID, purchase.PlategaURL, purchase.WataID, purchase.WataURL).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildInsert.ToSql()
	if err != nil {
		return 0, err
	}

	var id int64
	err = cr.pool.QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (cr *PurchaseRepository) FindByInvoiceTypeAndStatus(ctx context.Context, invoiceType InvoiceType, status PurchaseStatus) (*[]Purchase, error) {
	buildSelect := sq.Select("*").
		From("purchase").
		Where(sq.And{
			sq.Eq{"invoice_type": invoiceType},
			sq.Eq{"status": status},
		}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()

	purchases := []Purchase{}
	for rows.Next() {
		purchase, err := scanPurchase(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, *purchase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &purchases, nil
}

func (cr *PurchaseRepository) FindById(ctx context.Context, id int64) (*Purchase, error) {
	buildSelect := sq.Select("*").
		From("purchase").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, err
	}
	purchase, err := scanPurchase(cr.pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query purchase: %w", err)
	}

	return purchase, nil
}

func (p *PurchaseRepository) UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	if status, ok := updates["status"]; ok && fmt.Sprint(status) == string(PurchaseStatusPaid) {
		return errors.New("UpdateFields cannot mark a purchase paid; settle it with PurchaseRepository.Settle")
	}

	buildUpdate := sq.Update("purchase").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": id})

	for field, value := range updates {
		buildUpdate = buildUpdate.Set(field, value)
	}

	sql, args, err := buildUpdate.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no customer found with id: %d", id)
	}

	return nil
}

var ErrPurchaseNotFound = errors.New("purchase not found")

// Settle is the only way a purchase becomes paid. It holds the purchase row
// lock from the status check through apply to the status update, so
// concurrent or repeated deliveries of one payment run apply at most once.
// settled is false when the purchase was already paid or cancelled.
// apply must not query the pool: deliveries waiting on the lock hold
// connections, and a second acquisition can stall until the context ends.
func (pr *PurchaseRepository) Settle(
	ctx context.Context,
	id int64,
	apply func(ctx context.Context, purchase *Purchase) error,
) (purchase *Purchase, settled bool, err error) {
	tx, err := pr.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin settle: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sql, args, err := sq.Select("*").
		From("purchase").
		Where(sq.Eq{"id": id}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, false, err
	}
	purchase, err = scanPurchase(tx.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("%w: %d", ErrPurchaseNotFound, id)
	}
	if err != nil {
		return nil, false, fmt.Errorf("lock purchase: %w", err)
	}

	if purchase.Status == PurchaseStatusPaid || purchase.Status == PurchaseStatusCancel {
		return purchase, false, tx.Commit(ctx)
	}

	if err := apply(ctx, purchase); err != nil {
		return nil, false, err
	}

	paidAt := time.Now()
	if _, err := tx.Exec(ctx, "UPDATE purchase SET status = $1, paid_at = $2 WHERE id = $3", PurchaseStatusPaid, paidAt, id); err != nil {
		return nil, false, fmt.Errorf("mark purchase paid: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit settle: %w", err)
	}

	purchase.Status = PurchaseStatusPaid
	purchase.PaidAt = &paidAt
	return purchase, true, nil
}

func scanPurchase(row pgx.Row) (*Purchase, error) {
	p := &Purchase{}
	err := row.Scan(
		&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month,
		&p.PaidAt, &p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
		&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID,
		&p.PlategaID, &p.PlategaURL, &p.WataID, &p.WataURL,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func buildLatestActiveTributesQuery(customerIDs []int64) sq.SelectBuilder {
	return sq.
		Select("*").
		From("purchase").
		Where(sq.And{
			sq.Eq{"invoice_type": InvoiceTypeTribute},
			sq.Eq{"customer_id": customerIDs},
			sq.Expr("created_at = (SELECT MAX(created_at) FROM purchase p2 WHERE p2.customer_id = purchase.customer_id AND p2.invoice_type = ?)", InvoiceTypeTribute),
		}).
		Where(sq.NotEq{"status": PurchaseStatusCancel})
}

func (pr *PurchaseRepository) FindLatestActiveTributesByCustomerIDs(
	ctx context.Context,
	customerIDs []int64,
) (*[]Purchase, error) {
	if len(customerIDs) == 0 {
		empty := make([]Purchase, 0)
		return &empty, nil
	}

	builder := buildLatestActiveTributesQuery(customerIDs).PlaceholderFormat(sq.Dollar)

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query purchases: %w", err)
	}
	defer rows.Close()

	var purchases []Purchase
	for rows.Next() {
		p, err := scanPurchase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan purchase: %w", err)
		}
		purchases = append(purchases, *p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return &purchases, nil
}

func (pr *PurchaseRepository) FindByCustomerIDAndInvoiceTypeLast(
	ctx context.Context,
	customerID int64,
	invoiceType InvoiceType,
) (*Purchase, error) {

	query := sq.Select("*").
		From("purchase").
		Where(sq.And{
			sq.Eq{"customer_id": customerID},
			sq.Eq{"invoice_type": invoiceType},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	p, err := scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query purchase: %w", err)
	}

	return p, nil
}

func (pr *PurchaseRepository) FindSuccessfulPaidPurchaseByCustomer(ctx context.Context, customerID int64) (*Purchase, error) {
	query := sq.Select("*").
		From("purchase").
		Where(sq.And{
			sq.Eq{"customer_id": customerID},
			sq.Eq{"status": PurchaseStatusPaid},
			sq.Or{
				sq.Eq{"invoice_type": InvoiceTypeCrypto},
				sq.Eq{"invoice_type": InvoiceTypeYookasa},
				sq.Eq{"invoice_type": InvoiceTypePlatega},
				sq.Eq{"invoice_type": InvoiceTypePlategaSBP},
				sq.Eq{"invoice_type": InvoiceTypePlategaCards},
				sq.Eq{"invoice_type": InvoiceTypePlategaAcquiring},
				sq.Eq{"invoice_type": InvoiceTypePlategaWorldwide},
				sq.Eq{"invoice_type": InvoiceTypePlategaCrypto},
				sq.Eq{"invoice_type": InvoiceTypeWata},
			},
		}).
		OrderBy("paid_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	p, err := scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query purchase: %w", err)
	}

	return p, nil
}
