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

func (pr *PurchaseRepository) GetRevenueStats(ctx context.Context) (*stats.RevenueStats, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	startOfWeek := startOfDay.AddDate(0, 0, offset)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `
		SELECT 
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $1), 0) as today,
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $2), 0) as this_week,
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $3), 0) as this_month,
			COALESCE(SUM(amount), 0) as all_time,
			COALESCE(AVG(amount), 0) as avg_check
		FROM purchase
		WHERE status = $4
	`

	var s stats.RevenueStats
	err := pr.pool.QueryRow(ctx, query, startOfDay, startOfWeek, startOfMonth, PurchaseStatusPaid).Scan(
		&s.Today, &s.ThisWeek, &s.ThisMonth, &s.AllTime, &s.AvgCheck,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}

	return &s, nil
}

func (pr *PurchaseRepository) GetPaymentStats(ctx context.Context) (*stats.PaymentStats, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	totalsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE paid_at >= $1) as today
		FROM purchase
		WHERE status = $2
	`
	var s stats.PaymentStats
	err := pr.pool.QueryRow(ctx, totalsQuery, startOfDay, PurchaseStatusPaid).Scan(&s.TotalCount, &s.TodayCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment totals: %w", err)
	}

	currencyQuery := `
		SELECT 
			COALESCE(currency, 'unknown') as currency,
			COUNT(*) as count,
			COALESCE(SUM(amount), 0) as amount
		FROM purchase
		WHERE status = $1
		GROUP BY currency
		ORDER BY amount DESC
	`
	rows, err := pr.pool.Query(ctx, currencyQuery, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query currency stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item stats.CurrencyStat
		if err := rows.Scan(&item.Currency, &item.Count, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan currency stat: %w", err)
		}
		s.ByCurrency = append(s.ByCurrency, item)
	}

	typeQuery := `
		SELECT 
			COALESCE(invoice_type, 'unknown') as type,
			COUNT(*) as count,
			COALESCE(SUM(amount), 0) as amount
		FROM purchase
		WHERE status = $1
		GROUP BY invoice_type
		ORDER BY amount DESC
	`
	rows2, err := pr.pool.Query(ctx, typeQuery, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment type stats: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var item stats.PaymentTypeStat
		if err := rows2.Scan(&item.Type, &item.Count, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan payment type stat: %w", err)
		}
		s.ByPaymentType = append(s.ByPaymentType, item)
	}

	return &s, nil
}

func (pr *PurchaseRepository) GetDailyRevenue(ctx context.Context, days int) ([]stats.DailyRevenue, error) {
	query := `
		SELECT 
			TO_CHAR(days.day, 'YYYY-MM-DD') as date,
			COALESCE(SUM(p.amount), 0) as amount,
			COUNT(p.id) as count
		FROM generate_series(
			DATE_TRUNC('day', NOW() - INTERVAL '1 day' * $1),
			DATE_TRUNC('day', NOW()),
			'1 day'
		) AS days(day)
		LEFT JOIN purchase p ON DATE_TRUNC('day', p.paid_at) = days.day AND p.status = $2
		GROUP BY days.day
		ORDER BY days.day
	`

	rows, err := pr.pool.Query(ctx, query, days, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily revenue: %w", err)
	}
	defer rows.Close()

	var result []stats.DailyRevenue
	for rows.Next() {
		var item stats.DailyRevenue
		if err := rows.Scan(&item.Date, &item.Amount, &item.Count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}
