package database

import (
	"context"
	"errors"
	"fmt"
	"remnawave-tg-shop-bot/internal/stats"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type InvoiceType string

const (
	InvoiceTypeCrypto   InvoiceType = "crypto"
	InvoiceTypeYookasa  InvoiceType = "yookasa"
	InvoiceTypeTelegram InvoiceType = "telegram"
	InvoiceTypeTribute  InvoiceType = "tribute"
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
		Columns("amount", "customer_id", "month", "currency", "expire_at", "status", "invoice_type", "crypto_invoice_id", "crypto_invoice_url", "yookasa_url", "yookasa_id").
		Values(purchase.Amount, purchase.CustomerID, purchase.Month, purchase.Currency, purchase.ExpireAt, purchase.Status, purchase.InvoiceType, purchase.CryptoInvoiceID, purchase.CryptoInvoiceLink, purchase.YookasaURL, purchase.YookasaID).
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
		purchase := Purchase{}
		err = rows.Scan(
			&purchase.ID,
			&purchase.Amount,
			&purchase.CustomerID,
			&purchase.CreatedAt,
			&purchase.Month,
			&purchase.PaidAt,
			&purchase.Currency,
			&purchase.ExpireAt,
			&purchase.Status,
			&purchase.InvoiceType,
			&purchase.CryptoInvoiceID,
			&purchase.CryptoInvoiceLink,
			&purchase.YookasaURL,
			&purchase.YookasaID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, purchase)
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
	purchase := &Purchase{}

	err = cr.pool.QueryRow(ctx, sql, args...).Scan(
		&purchase.ID,
		&purchase.Amount,
		&purchase.CustomerID,
		&purchase.CreatedAt,
		&purchase.Month,
		&purchase.PaidAt,
		&purchase.Currency,
		&purchase.ExpireAt,
		&purchase.Status,
		&purchase.InvoiceType,
		&purchase.CryptoInvoiceID,
		&purchase.CryptoInvoiceLink,
		&purchase.YookasaURL,
		&purchase.YookasaID,
	)

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

func (pr *PurchaseRepository) MarkAsPaid(ctx context.Context, purchaseID int64) error {
	currentTime := time.Now()

	updates := map[string]interface{}{
		"status":  PurchaseStatusPaid,
		"paid_at": currentTime,
	}

	return pr.UpdateFields(ctx, purchaseID, updates)
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
		var p Purchase
		if err := rows.Scan(
			&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month,
			&p.PaidAt, &p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
			&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID,
		); err != nil {
			return nil, fmt.Errorf("scan purchase: %w", err)
		}
		purchases = append(purchases, p)
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

	p := &Purchase{}
	err = pr.pool.QueryRow(ctx, sql, args...).Scan(
		&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month,
		&p.PaidAt, &p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
		&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID,
	)
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
			},
		}).
		OrderBy("paid_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	p := &Purchase{}
	err = pr.pool.QueryRow(ctx, sql, args...).Scan(
		&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month,
		&p.PaidAt, &p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
		&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query purchase: %w", err)
	}

	return p, nil
}

func (pr *PurchaseRepository) GetTotalAmountByDateRange(ctx context.Context, start, end time.Time) (float64, error) {
	query := `
        SELECT COALESCE(SUM(amount), 0)
        FROM purchase
        WHERE status = $1 AND paid_at >= $2 AND paid_at <= $3
    `
	var total float64
	err := pr.pool.QueryRow(ctx, query, PurchaseStatusPaid, start, end).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total amount: %w", err)
	}
	return total, nil
}

func (pr *PurchaseRepository) GetMonthlyGrowthLastYear(ctx context.Context) ([]stats.MonthlyGrowth, error) {
	query := `
        SELECT 
            TO_CHAR(months.month, 'Mon YYYY') as month,
            COALESCE(SUM(p.amount), 0) as amount
        FROM generate_series(
            DATE_TRUNC('month', NOW() - INTERVAL '11 months'),
            DATE_TRUNC('month', NOW()),
            '1 month'
        ) AS months(month)
        LEFT JOIN purchase p ON DATE_TRUNC('month', p.paid_at) = months.month AND p.status = $1
        GROUP BY months.month
        ORDER BY months.month
    `
	rows, err := pr.pool.Query(ctx, query, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly growth: %w", err)
	}
	defer rows.Close()

	var result []stats.MonthlyGrowth
	for rows.Next() {
		var item stats.MonthlyGrowth
		err := rows.Scan(&item.Month, &item.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly growth row: %w", err)
		}
		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly growth rows: %w", err)
	}

	return result, nil
}

func (pr *PurchaseRepository) FindByCustomerID(ctx context.Context, customerID int64) ([]Purchase, error) {
	buildSelect := sq.Select("*").
		From("purchase").
		Where(sq.Eq{"customer_id": customerID}).
		OrderBy("created_at DESC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()

	var purchases []Purchase
	for rows.Next() {
		var purchase Purchase
		err = rows.Scan(
			&purchase.ID,
			&purchase.Amount,
			&purchase.CustomerID,
			&purchase.CreatedAt,
			&purchase.Month,
			&purchase.PaidAt,
			&purchase.Currency,
			&purchase.ExpireAt,
			&purchase.Status,
			&purchase.InvoiceType,
			&purchase.CryptoInvoiceID,
			&purchase.CryptoInvoiceLink,
			&purchase.YookasaURL,
			&purchase.YookasaID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, purchase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return purchases, nil
}
