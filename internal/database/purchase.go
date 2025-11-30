package database

import (
	"context"
	"errors"
	"fmt"
	"remnawave-tg-shop-bot/internal/stats"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type InvoiceType string

const (
	InvoiceTypeCrypto   InvoiceType = "crypto"
	InvoiceTypeYookassa InvoiceType = "yookassa"
	InvoiceTypeTelegram InvoiceType = "telegram"
	InvoiceTypeTribute  InvoiceType = "tribute"
)

type PurchaseStatus string

const (
	PurchaseStatusNew        PurchaseStatus = "new"
	PurchaseStatusPending    PurchaseStatus = "pending"
	PurchaseStatusProcessing PurchaseStatus = "processing"
	PurchaseStatusPaid       PurchaseStatus = "paid"
	PurchaseStatusCancel     PurchaseStatus = "cancel"
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
	PlanID            *int64         `db:"plan_id"`
}

var purchaseColumns = []string{
	"id", "amount", "customer_id", "created_at", "month", "paid_at",
	"currency", "expire_at", "status", "invoice_type",
	"crypto_invoice_id", "crypto_invoice_url", "yookasa_url", "yookasa_id", "plan_id",
}

type PurchaseRepository struct {
	pool *pgxpool.Pool
}

func NewPurchaseRepository(pool *pgxpool.Pool) *PurchaseRepository {
	return &PurchaseRepository{pool: pool}
}

func scanPurchase(row pgx.Row) (*Purchase, error) {
	p := &Purchase{}
	err := row.Scan(
		&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month, &p.PaidAt,
		&p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
		&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID, &p.PlanID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func scanPurchaseRows(rows pgx.Rows) ([]Purchase, error) {
	var purchases []Purchase
	for rows.Next() {
		p := Purchase{}
		err := rows.Scan(
			&p.ID, &p.Amount, &p.CustomerID, &p.CreatedAt, &p.Month, &p.PaidAt,
			&p.Currency, &p.ExpireAt, &p.Status, &p.InvoiceType,
			&p.CryptoInvoiceID, &p.CryptoInvoiceLink, &p.YookasaURL, &p.YookasaID, &p.PlanID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return purchases, nil
}

func (pr *PurchaseRepository) Create(ctx context.Context, purchase *Purchase) (int64, error) {
	query := sq.Insert("purchase").
		Columns("amount", "customer_id", "month", "currency", "expire_at", "status", "invoice_type", "crypto_invoice_id", "crypto_invoice_url", "yookasa_url", "yookasa_id", "plan_id").
		Values(purchase.Amount, purchase.CustomerID, purchase.Month, purchase.Currency, purchase.ExpireAt, purchase.Status, purchase.InvoiceType, purchase.CryptoInvoiceID, purchase.CryptoInvoiceLink, purchase.YookasaURL, purchase.YookasaID, purchase.PlanID).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, err
	}

	var id int64
	err = pr.pool.QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (pr *PurchaseRepository) FindById(ctx context.Context, id int64) (*Purchase, error) {
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	return scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
}

func (pr *PurchaseRepository) FindByInvoiceTypeAndStatus(ctx context.Context, invoiceType InvoiceType, status PurchaseStatus) ([]Purchase, error) {
	// Exclude purchases that are currently being processed to avoid race conditions
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.And{
			sq.Eq{"invoice_type": invoiceType},
			sq.Eq{"status": status},
			sq.NotEq{"status": PurchaseStatusProcessing},
		}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()
	return scanPurchaseRows(rows)
}

func (pr *PurchaseRepository) FindByCustomerID(ctx context.Context, customerID int64) ([]Purchase, error) {
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.Eq{"customer_id": customerID}).
		OrderBy("created_at DESC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()
	return scanPurchaseRows(rows)
}

func (pr *PurchaseRepository) FindByCustomerIDAndInvoiceTypeLast(ctx context.Context, customerID int64, invoiceType InvoiceType) (*Purchase, error) {
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.And{sq.Eq{"customer_id": customerID}, sq.Eq{"invoice_type": invoiceType}}).
		OrderBy("created_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
}

func (pr *PurchaseRepository) FindSuccessfulPaidPurchaseByCustomer(ctx context.Context, customerID int64) (*Purchase, error) {
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.And{
			sq.Eq{"customer_id": customerID},
			sq.Eq{"status": PurchaseStatusPaid},
			sq.Or{sq.Eq{"invoice_type": InvoiceTypeCrypto}, sq.Eq{"invoice_type": InvoiceTypeYookassa}},
		}).
		OrderBy("paid_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
}

func (pr *PurchaseRepository) FindLastPaidPurchaseWithPlan(ctx context.Context, customerID int64) (*Purchase, error) {
	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.And{
			sq.Eq{"customer_id": customerID},
			sq.Eq{"status": PurchaseStatusPaid},
			sq.NotEq{"plan_id": nil},
		}).
		OrderBy("paid_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return scanPurchase(pr.pool.QueryRow(ctx, sql, args...))
}

func (pr *PurchaseRepository) FindLatestActiveTributesByCustomerIDs(ctx context.Context, customerIDs []int64) ([]Purchase, error) {
	if len(customerIDs) == 0 {
		return []Purchase{}, nil
	}

	query := sq.Select(purchaseColumns...).
		From("purchase").
		Where(sq.And{
			sq.Eq{"invoice_type": InvoiceTypeTribute},
			sq.Eq{"customer_id": customerIDs},
			sq.Expr("created_at = (SELECT MAX(created_at) FROM purchase p2 WHERE p2.customer_id = purchase.customer_id AND p2.invoice_type = ?)", InvoiceTypeTribute),
		}).
		Where(sq.NotEq{"status": PurchaseStatusCancel}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query purchases: %w", err)
	}
	defer rows.Close()
	return scanPurchaseRows(rows)
}

func (pr *PurchaseRepository) UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := sq.Update("purchase").PlaceholderFormat(sq.Dollar).Where(sq.Eq{"id": id})
	for field, value := range updates {
		query = query.Set(field, value)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := pr.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update purchase: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no purchase found with id: %d", id)
	}
	return nil
}

func (pr *PurchaseRepository) MarkAsPaid(ctx context.Context, purchaseID int64) error {
	return pr.UpdateFields(ctx, purchaseID, map[string]interface{}{
		"status":  PurchaseStatusPaid,
		"paid_at": time.Now(),
	})
}

// LockForProcessing atomically locks a purchase for processing.
// Returns the purchase if successfully locked, nil if already locked/processed by another worker.
// This prevents race conditions when multiple workers try to process the same purchase.
func (pr *PurchaseRepository) LockForProcessing(ctx context.Context, purchaseID int64) (*Purchase, error) {
	query := `
		UPDATE purchase SET status = $1
		WHERE id = $2 AND status = $3
		RETURNING ` + strings.Join(purchaseColumns, ", ")

	row := pr.pool.QueryRow(ctx, query, PurchaseStatusProcessing, purchaseID, PurchaseStatusPending)
	return scanPurchase(row)
}

// UnlockPurchase releases the lock if processing failed, returning to pending status
func (pr *PurchaseRepository) UnlockPurchase(ctx context.Context, purchaseID int64) error {
	_, err := pr.pool.Exec(ctx,
		"UPDATE purchase SET status = $1 WHERE id = $2 AND status = $3",
		PurchaseStatusPending, purchaseID, PurchaseStatusProcessing,
	)
	return err
}

func (pr *PurchaseRepository) CountByPlanID(ctx context.Context, planID int64) (int64, error) {
	var count int64
	err := pr.pool.QueryRow(ctx, "SELECT COUNT(*) FROM purchase WHERE plan_id = $1", planID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count purchases: %w", err)
	}
	return count, nil
}

// Stats methods

func (pr *PurchaseRepository) GetTotalAmountByDateRange(ctx context.Context, start, end time.Time) (float64, error) {
	var total float64
	err := pr.pool.QueryRow(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM purchase WHERE status = $1 AND paid_at >= $2 AND paid_at <= $3",
		PurchaseStatusPaid, start, end,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total amount: %w", err)
	}
	return total, nil
}

func (pr *PurchaseRepository) GetMonthlyGrowthLastYear(ctx context.Context) ([]stats.MonthlyGrowth, error) {
	query := `
		SELECT TO_CHAR(months.month, 'Mon YYYY') as month, COALESCE(SUM(p.amount), 0) as amount
		FROM generate_series(DATE_TRUNC('month', NOW() - INTERVAL '11 months'), DATE_TRUNC('month', NOW()), '1 month') AS months(month)
		LEFT JOIN purchase p ON DATE_TRUNC('month', p.paid_at) = months.month AND p.status = $1
		GROUP BY months.month ORDER BY months.month`

	rows, err := pr.pool.Query(ctx, query, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly growth: %w", err)
	}
	defer rows.Close()

	var result []stats.MonthlyGrowth
	for rows.Next() {
		var item stats.MonthlyGrowth
		if err := rows.Scan(&item.Month, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
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
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $1), 0),
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $2), 0),
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $3), 0),
			COALESCE(SUM(amount), 0),
			COALESCE(AVG(amount), 0)
		FROM purchase WHERE status = $4`

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

	var s stats.PaymentStats
	err := pr.pool.QueryRow(ctx,
		"SELECT COUNT(*), COUNT(*) FILTER (WHERE paid_at >= $1) FROM purchase WHERE status = $2",
		startOfDay, PurchaseStatusPaid,
	).Scan(&s.TotalCount, &s.TodayCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment totals: %w", err)
	}

	// By currency
	rows, err := pr.pool.Query(ctx,
		"SELECT COALESCE(currency, 'unknown'), COUNT(*), COALESCE(SUM(amount), 0) FROM purchase WHERE status = $1 GROUP BY currency ORDER BY SUM(amount) DESC",
		PurchaseStatusPaid,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query currency stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item stats.CurrencyStat
		if err := rows.Scan(&item.Currency, &item.Count, &item.Amount); err != nil {
			return nil, err
		}
		s.ByCurrency = append(s.ByCurrency, item)
	}

	// By payment type
	rows2, err := pr.pool.Query(ctx,
		"SELECT COALESCE(invoice_type, 'unknown'), COUNT(*), COALESCE(SUM(amount), 0) FROM purchase WHERE status = $1 GROUP BY invoice_type ORDER BY SUM(amount) DESC",
		PurchaseStatusPaid,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment type stats: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var item stats.PaymentTypeStat
		if err := rows2.Scan(&item.Type, &item.Count, &item.Amount); err != nil {
			return nil, err
		}
		s.ByPaymentType = append(s.ByPaymentType, item)
	}

	return &s, nil
}

func (pr *PurchaseRepository) GetDailyRevenue(ctx context.Context, days int) ([]stats.DailyRevenue, error) {
	query := `
		SELECT TO_CHAR(days.day, 'YYYY-MM-DD'), COALESCE(SUM(p.amount), 0), COUNT(p.id)
		FROM generate_series(DATE_TRUNC('day', NOW() - INTERVAL '1 day' * $1), DATE_TRUNC('day', NOW()), '1 day') AS days(day)
		LEFT JOIN purchase p ON DATE_TRUNC('day', p.paid_at) = days.day AND p.status = $2
		GROUP BY days.day ORDER BY days.day`

	rows, err := pr.pool.Query(ctx, query, days, PurchaseStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily revenue: %w", err)
	}
	defer rows.Close()

	var result []stats.DailyRevenue
	for rows.Next() {
		var item stats.DailyRevenue
		if err := rows.Scan(&item.Date, &item.Amount, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
