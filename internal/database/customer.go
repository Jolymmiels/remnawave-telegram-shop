package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"remnawave-tg-shop-bot/internal/stats"
	"remnawave-tg-shop-bot/utils"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(poll *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: poll}
}

type Customer struct {
	ID                    int64      `db:"id" json:"id"`
	TelegramID            int64      `db:"telegram_id" json:"telegram_id"`
	ExpireAt              *time.Time `db:"expire_at" json:"expire_at"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	SubscriptionLink      *string    `db:"subscription_link" json:"subscription_link"`
	Language              string     `db:"language" json:"language"`
	IsBlocked             bool       `db:"is_blocked" json:"is_blocked"`
	IsBlockedByUser       bool       `db:"is_blocked_by_user" json:"is_blocked_by_user"`
	TrialUsed             bool       `db:"trial_used" json:"trial_used"`
	PaymentMethodID       *string    `db:"payment_method_id" json:"payment_method_id"`
	AutopayEnabled        bool       `db:"autopay_enabled" json:"autopay_enabled"`
	AutopayPlanID         *int64     `db:"autopay_plan_id" json:"autopay_plan_id"`
	AutopayMonths         int        `db:"autopay_months" json:"autopay_months"`
	AutopayFailedAttempts int        `db:"autopay_failed_attempts" json:"autopay_failed_attempts"`
	TgUsername            *string    `db:"tg_username" json:"tg_username"`
	TgFirstName           *string    `db:"tg_first_name" json:"tg_first_name"`
	TgLastName            *string    `db:"tg_last_name" json:"tg_last_name"`
}

var customerColumns = []string{
	"id", "telegram_id", "expire_at", "created_at", "subscription_link",
	"language", "is_blocked", "is_blocked_by_user", "trial_used", "payment_method_id", "autopay_enabled", "autopay_plan_id", "autopay_months", "autopay_failed_attempts",
	"tg_username", "tg_first_name", "tg_last_name",
}

func scanCustomer(row interface{ Scan(dest ...any) error }) (*Customer, error) {
	var c Customer
	err := row.Scan(
		&c.ID, &c.TelegramID, &c.ExpireAt, &c.CreatedAt, &c.SubscriptionLink,
		&c.Language, &c.IsBlocked, &c.IsBlockedByUser, &c.TrialUsed, &c.PaymentMethodID, &c.AutopayEnabled, &c.AutopayPlanID, &c.AutopayMonths, &c.AutopayFailedAttempts,
		&c.TgUsername, &c.TgFirstName, &c.TgLastName,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (cr *CustomerRepository) FindByExpirationRange(ctx context.Context, startDate, endDate time.Time) (*[]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(
			sq.And{
				sq.NotEq{"expire_at": nil},
				sq.GtOrEq{"expire_at": startDate},
				sq.LtOrEq{"expire_at": endDate},
			},
		).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers by expiration range: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindById(ctx context.Context, id int64) (*Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	customer, err := scanCustomer(cr.pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return customer, nil
}

func (cr *CustomerRepository) FindByTelegramId(ctx context.Context, telegramId int64) (*Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.Eq{"telegram_id": telegramId}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	customer, err := scanCustomer(cr.pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return customer, nil
}

func (cr *CustomerRepository) Create(ctx context.Context, customer *Customer) (*Customer, error) {
	return cr.FindOrCreate(ctx, customer)
}

func (cr *CustomerRepository) FindOrCreate(ctx context.Context, customer *Customer) (*Customer, error) {
	query := `
		INSERT INTO customer (telegram_id, expire_at, language, tg_username, tg_first_name, tg_last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (telegram_id) DO UPDATE SET 
			tg_username = COALESCE(EXCLUDED.tg_username, customer.tg_username),
			tg_first_name = COALESCE(EXCLUDED.tg_first_name, customer.tg_first_name),
			tg_last_name = COALESCE(EXCLUDED.tg_last_name, customer.tg_last_name)
		RETURNING id, telegram_id, expire_at, created_at, subscription_link, language, is_blocked, is_blocked_by_user, trial_used, 
			payment_method_id, autopay_enabled, autopay_plan_id, autopay_months, autopay_failed_attempts,
			tg_username, tg_first_name, tg_last_name
	`

	row := cr.pool.QueryRow(ctx, query, customer.TelegramID, customer.ExpireAt, customer.Language, 
		customer.TgUsername, customer.TgFirstName, customer.TgLastName)
	result, err := scanCustomer(row)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create customer: %w", err)
	}

	slog.Info("user found or created in bot database", "telegramId", utils.MaskHalfInt64(result.TelegramID))
	return result, nil
}

func (cr *CustomerRepository) UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	buildUpdate := sq.Update("customer").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": id})

	for field, value := range updates {
		buildUpdate = buildUpdate.Set(field, value)
	}

	sql, args, err := buildUpdate.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	tx, err := cr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	result, err := cr.pool.Exec(ctx, sql, args...)
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
		return fmt.Errorf("failed to update customer: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no customer found with id: %s", utils.MaskHalfInt64(id))
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (cr *CustomerRepository) FindByTelegramIds(ctx context.Context, telegramIDs []int64) ([]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.Eq{"telegram_id": telegramIDs}).
		PlaceholderFormat(sq.Dollar)

	sqlStr, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return customers, nil
}

func (cr *CustomerRepository) CreateBatch(ctx context.Context, customers []Customer) error {
	if len(customers) == 0 {
		return nil
	}
	builder := sq.Insert("customer").
		Columns("telegram_id", "expire_at", "language", "subscription_link").
		PlaceholderFormat(sq.Dollar)
	for _, cust := range customers {
		builder = builder.Values(cust.TelegramID, cust.ExpireAt, cust.Language, cust.SubscriptionLink)
	}
	sqlStr, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build batch insert query: %w", err)
	}

	tx, err := cr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = cr.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (cr *CustomerRepository) UpdateBatch(ctx context.Context, customers []Customer) error {
	if len(customers) == 0 {
		return nil
	}
	query := "UPDATE customer SET expire_at = c.expire_at, subscription_link = c.subscription_link FROM (VALUES "
	var args []interface{}
	for i, cust := range customers {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d::bigint, $%d::timestamp, $%d::text)", i*3+1, i*3+2, i*3+3)
		args = append(args, cust.TelegramID, cust.ExpireAt, cust.SubscriptionLink)
	}
	query += ") AS c(telegram_id, expire_at, subscription_link) WHERE customer.telegram_id = c.telegram_id"

	tx, err := cr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	_, err = cr.pool.Exec(ctx, query, args...)
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
		return fmt.Errorf("failed to execute batch update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (cr *CustomerRepository) DeleteByTelegramId(ctx context.Context, telegramID int64) error {
	buildDelete := sq.Delete("customer").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"telegram_id": telegramID})

	sqlStr, args, err := buildDelete.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = cr.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	return nil
}

func (cr *CustomerRepository) DeleteByNotInTelegramIds(ctx context.Context, telegramIDs []int64) error {
	var buildDelete sq.DeleteBuilder
	if len(telegramIDs) == 0 {
		buildDelete = sq.Delete("customer")
	} else {
		buildDelete = sq.Delete("customer").
			PlaceholderFormat(sq.Dollar).
			Where(sq.NotEq{"telegram_id": telegramIDs})
	}

	sqlStr, args, err := buildDelete.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = cr.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to delete customers: %w", err)
	}

	return nil

}

func (cr *CustomerRepository) SetBlocked(ctx context.Context, telegramID int64, blocked bool) error {
	query := `UPDATE customer SET is_blocked = $1 WHERE telegram_id = $2`
	_, err := cr.pool.Exec(ctx, query, blocked, telegramID)
	return err
}

func (cr *CustomerRepository) SetBlockedBatch(ctx context.Context, telegramIDs []int64, blocked bool) error {
	if len(telegramIDs) == 0 {
		return nil
	}
	query := `UPDATE customer SET is_blocked = $1 WHERE telegram_id = ANY($2)`
	_, err := cr.pool.Exec(ctx, query, blocked, telegramIDs)
	return err
}

func (cr *CustomerRepository) SetBlockedByUser(ctx context.Context, telegramID int64, blocked bool) error {
	query := `UPDATE customer SET is_blocked_by_user = $1 WHERE telegram_id = $2`
	_, err := cr.pool.Exec(ctx, query, blocked, telegramID)
	return err
}

func (cr *CustomerRepository) SetBlockedByUserBatch(ctx context.Context, telegramIDs []int64, blocked bool) error {
	if len(telegramIDs) == 0 {
		return nil
	}
	query := `UPDATE customer SET is_blocked_by_user = $1 WHERE telegram_id = ANY($2)`
	_, err := cr.pool.Exec(ctx, query, blocked, telegramIDs)
	return err
}

type UserGrowthStats struct {
	NewUsersLastMonth int64 `json:"new_users_last_month"`
	TotalUsers        int64 `json:"total_users"`
}

func (cr *CustomerRepository) GetUserGrowthStats(ctx context.Context) (*UserGrowthStats, error) {
	stats := &UserGrowthStats{}

	now := time.Now()
	startOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	countNewQuery := `SELECT COUNT(*) FROM customer WHERE created_at >= $1 AND created_at < $2`
	startOfLastMonth := startOfCurrentMonth.AddDate(0, -1, 0)
	endOfLastMonth := startOfCurrentMonth

	err := cr.pool.QueryRow(ctx, countNewQuery, startOfLastMonth, endOfLastMonth).Scan(&stats.NewUsersLastMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to count new users in the last month: %w", err)
	}

	countTotalQuery := `SELECT COUNT(*) FROM customer`
	err = cr.pool.QueryRow(ctx, countTotalQuery).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	return stats, nil
}

type CustomerSortField string

const (
	SortByDate      CustomerSortField = "date"
	SortBySpent     CustomerSortField = "spent"
	SortByReferrals CustomerSortField = "referrals"
)

type CustomerSortOrder string

const (
	SortAsc  CustomerSortOrder = "asc"
	SortDesc CustomerSortOrder = "desc"
)

type CustomerStatusFilter string

const (
	StatusAll            CustomerStatusFilter = ""
	StatusActive         CustomerStatusFilter = "active"
	StatusExpired        CustomerStatusFilter = "expired"
	StatusNoSubscription CustomerStatusFilter = "no_subscription"
)

type CustomerWithStats struct {
	Customer
	TotalSpent     float64 `db:"total_spent" json:"total_spent"`
	ReferralsCount int     `db:"referrals_count" json:"referrals_count"`
	PaymentsCount  int     `db:"payments_count" json:"payments_count"`
}

type CustomerSearchParams struct {
	Query     string
	Status    CustomerStatusFilter
	SortBy    CustomerSortField
	SortOrder CustomerSortOrder
	Limit     int
	Offset    int
}

func (cr *CustomerRepository) FindAllSorted(ctx context.Context, params CustomerSearchParams) ([]CustomerWithStats, int, error) {
	orderDir := "DESC"
	if params.SortOrder == SortAsc {
		orderDir = "ASC"
	}

	var orderClause string
	switch params.SortBy {
	case SortBySpent:
		orderClause = fmt.Sprintf("total_spent %s", orderDir)
	case SortByReferrals:
		orderClause = fmt.Sprintf("referrals_count %s", orderDir)
	default:
		orderClause = fmt.Sprintf("c.created_at %s", orderDir)
	}

	// Build WHERE clause
	var whereConditions []string
	var args []interface{}
	argNum := 1

	// Search by telegram_id, username, first_name, last_name (partial match)
	if params.Query != "" {
		searchCondition := fmt.Sprintf(`(
			CAST(c.telegram_id AS TEXT) LIKE $%d 
			OR LOWER(c.tg_username) LIKE LOWER($%d)
			OR LOWER(c.tg_first_name) LIKE LOWER($%d)
			OR LOWER(c.tg_last_name) LIKE LOWER($%d)
		)`, argNum, argNum, argNum, argNum)
		whereConditions = append(whereConditions, searchCondition)
		args = append(args, "%"+params.Query+"%")
		argNum++
	}

	// Filter by status
	switch params.Status {
	case StatusActive:
		whereConditions = append(whereConditions, "c.expire_at IS NOT NULL AND c.expire_at > NOW()")
	case StatusExpired:
		whereConditions = append(whereConditions, "c.expire_at IS NOT NULL AND c.expire_at <= NOW()")
	case StatusNoSubscription:
		whereConditions = append(whereConditions, "c.expire_at IS NULL")
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT 
			c.id, c.telegram_id, c.expire_at, c.created_at, c.subscription_link,
			c.language, c.is_blocked, c.is_blocked_by_user, c.trial_used, c.payment_method_id, c.autopay_enabled, c.autopay_plan_id, c.autopay_months, c.autopay_failed_attempts,
			c.tg_username, c.tg_first_name, c.tg_last_name,
			COALESCE(ps.total_spent, 0) as total_spent,
			COALESCE(ps.payments_count, 0) as payments_count,
			COALESCE(rc.referrals_count, 0) as referrals_count
		FROM customer c
		LEFT JOIN (
			SELECT customer_id, SUM(amount) as total_spent, COUNT(*) as payments_count
			FROM purchase
			WHERE status = 'paid'
			GROUP BY customer_id
		) ps ON ps.customer_id = c.id
		LEFT JOIN (
			SELECT referrer_id, COUNT(*) as referrals_count
			FROM referral
			GROUP BY referrer_id
		) rc ON rc.referrer_id = c.telegram_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argNum, argNum+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := cr.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query customers sorted: %w", err)
	}
	defer rows.Close()

	var customers []CustomerWithStats
	for rows.Next() {
		var c CustomerWithStats
		err := rows.Scan(
			&c.ID, &c.TelegramID, &c.ExpireAt, &c.CreatedAt, &c.SubscriptionLink,
			&c.Language, &c.IsBlocked, &c.IsBlockedByUser, &c.TrialUsed, &c.PaymentMethodID, &c.AutopayEnabled, &c.AutopayPlanID, &c.AutopayMonths, &c.AutopayFailedAttempts,
			&c.TgUsername, &c.TgFirstName, &c.TgLastName,
			&c.TotalSpent, &c.PaymentsCount, &c.ReferralsCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	// Get total count with same filters
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customer c %s", whereClause)
	countArgs := args[:len(args)-2] // Remove limit and offset

	var total int
	err = cr.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count customers: %w", err)
	}

	return customers, total, nil
}

func (cr *CustomerRepository) FindAll(ctx context.Context) (*[]Customer, error) {
	return cr.FindAllWithLanguage(ctx, "")
}

func (cr *CustomerRepository) FindAllWithLanguage(ctx context.Context, language string) (*[]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"is_blocked_by_user": false})

	if language != "" {
		buildSelect = buildSelect.Where(sq.Eq{"language": language})
	}

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query all customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindNonExpired(ctx context.Context) (*[]Customer, error) {
	return cr.FindNonExpiredWithLanguage(ctx, "")
}

func (cr *CustomerRepository) FindNonExpiredWithLanguage(ctx context.Context, language string) (*[]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.Gt{"expire_at": time.Now()}).
		Where(sq.Eq{"is_blocked_by_user": false}).
		PlaceholderFormat(sq.Dollar)

	if language != "" {
		buildSelect = buildSelect.Where(sq.Eq{"language": language})
	}

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query non-expired customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindExpired(ctx context.Context) (*[]Customer, error) {
	return cr.FindExpiredWithLanguage(ctx, "")
}

func (cr *CustomerRepository) FindExpiredWithLanguage(ctx context.Context, language string) (*[]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.LtOrEq{"expire_at": time.Now()}).
		Where(sq.Eq{"is_blocked_by_user": false}).
		PlaceholderFormat(sq.Dollar)

	if language != "" {
		buildSelect = buildSelect.Where(sq.Eq{"language": language})
	}

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindNoSubscription(ctx context.Context) (*[]Customer, error) {
	return cr.FindNoSubscriptionWithLanguage(ctx, "")
}

func (cr *CustomerRepository) FindNoSubscriptionWithLanguage(ctx context.Context, language string) (*[]Customer, error) {
	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.Eq{"expire_at": nil}).
		Where(sq.Eq{"is_blocked_by_user": false}).
		PlaceholderFormat(sq.Dollar)

	if language != "" {
		buildSelect = buildSelect.Where(sq.Eq{"language": language})
	}

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers without subscription: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) GetDistinctLanguages(ctx context.Context) ([]string, error) {
	query := sq.Select("DISTINCT language").
		From("customer").
		Where(sq.NotEq{"language": ""}).
		OrderBy("language").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var languages []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, lang)
	}

	return languages, nil
}

func (cr *CustomerRepository) GetUserStats(ctx context.Context) (*stats.UserStats, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Start of week (Monday)
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	startOfWeek := startOfDay.AddDate(0, 0, offset)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE expire_at > NOW()) as active,
			COUNT(*) FILTER (WHERE expire_at <= NOW() OR expire_at IS NULL) as expired,
			COUNT(*) FILTER (WHERE is_blocked = true) as blocked,
			COUNT(*) FILTER (WHERE created_at >= $1) as new_today,
			COUNT(*) FILTER (WHERE created_at >= $2) as new_this_week,
			COUNT(*) FILTER (WHERE created_at >= $3) as new_this_month
		FROM customer
	`

	var s stats.UserStats
	err := cr.pool.QueryRow(ctx, query, startOfDay, startOfWeek, startOfMonth).Scan(
		&s.Total, &s.Active, &s.Expired, &s.Blocked, &s.NewToday, &s.NewThisWeek, &s.NewThisMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &s, nil
}

func (cr *CustomerRepository) GetDailyUserGrowth(ctx context.Context, days int) ([]stats.DailyGrowth, error) {
	query := `
		SELECT 
			TO_CHAR(days.day, 'YYYY-MM-DD') as date,
			COUNT(c.id) as count
		FROM generate_series(
			DATE_TRUNC('day', NOW() - INTERVAL '1 day' * $1),
			DATE_TRUNC('day', NOW()),
			'1 day'
		) AS days(day)
		LEFT JOIN customer c ON DATE_TRUNC('day', c.created_at) = days.day
		GROUP BY days.day
		ORDER BY days.day
	`

	rows, err := cr.pool.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily growth: %w", err)
	}
	defer rows.Close()

	var result []stats.DailyGrowth
	for rows.Next() {
		var item stats.DailyGrowth
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// FindCustomersWithExpiringAutopay finds customers whose subscription expires within the given days
// and have autopay enabled with a valid payment method
func (cr *CustomerRepository) FindCustomersWithExpiringAutopay(ctx context.Context, daysBeforeExpiry int) (*[]Customer, error) {
	now := time.Now()
	expiryThreshold := now.AddDate(0, 0, daysBeforeExpiry)

	buildSelect := sq.Select(customerColumns...).
		From("customer").
		Where(sq.And{
			sq.NotEq{"payment_method_id": nil},
			sq.Eq{"autopay_enabled": true},
			sq.NotEq{"autopay_plan_id": nil},
			sq.Gt{"expire_at": now},
			sq.LtOrEq{"expire_at": expiryThreshold},
		}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers with expiring autopay: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

// SetPaymentMethod saves the payment method ID for autopay and resets failed attempts counter
func (cr *CustomerRepository) SetPaymentMethod(ctx context.Context, customerID int64, paymentMethodID string, autopayPlanID int64, months int) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"payment_method_id":       paymentMethodID,
		"autopay_enabled":         true,
		"autopay_plan_id":         autopayPlanID,
		"autopay_months":          months,
		"autopay_failed_attempts": 0,
	})
}

// DisableAutopay disables autopay for a customer
func (cr *CustomerRepository) DisableAutopay(ctx context.Context, customerID int64) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"autopay_enabled": false,
	})
}

// DisableAutopayByTelegramID disables autopay for a customer by their telegram ID
func (cr *CustomerRepository) DisableAutopayByTelegramID(ctx context.Context, telegramID int64) error {
	customer, err := cr.FindByTelegramId(ctx, telegramID)
	if err != nil {
		return err
	}
	if customer == nil {
		return fmt.Errorf("customer not found")
	}
	return cr.DisableAutopay(ctx, customer.ID)
}

// IncrementAutopayFailedAttempts increments the failed attempts counter and returns the new value
func (cr *CustomerRepository) IncrementAutopayFailedAttempts(ctx context.Context, customerID int64) (int, error) {
	query := `UPDATE customer SET autopay_failed_attempts = autopay_failed_attempts + 1 WHERE id = $1 RETURNING autopay_failed_attempts`
	var newCount int
	err := cr.pool.QueryRow(ctx, query, customerID).Scan(&newCount)
	if err != nil {
		return 0, fmt.Errorf("failed to increment autopay failed attempts: %w", err)
	}
	return newCount, nil
}

// ResetAutopayFailedAttempts resets the failed attempts counter to 0
func (cr *CustomerRepository) ResetAutopayFailedAttempts(ctx context.Context, customerID int64) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"autopay_failed_attempts": 0,
	})
}

// DisableAutopayAndReset disables autopay and resets the failed attempts counter
func (cr *CustomerRepository) DisableAutopayAndReset(ctx context.Context, customerID int64) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"autopay_enabled":         false,
		"autopay_failed_attempts": 0,
	})
}

// DeletePaymentMethod removes the payment method and disables autopay
func (cr *CustomerRepository) DeletePaymentMethod(ctx context.Context, customerID int64) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"payment_method_id":       nil,
		"autopay_enabled":         false,
		"autopay_plan_id":         nil,
		"autopay_months":          0,
		"autopay_failed_attempts": 0,
	})
}

// DeletePaymentMethodByTelegramID removes the payment method by telegram ID
func (cr *CustomerRepository) DeletePaymentMethodByTelegramID(ctx context.Context, telegramID int64) error {
	customer, err := cr.FindByTelegramId(ctx, telegramID)
	if err != nil {
		return err
	}
	if customer == nil {
		return nil
	}
	return cr.DeletePaymentMethod(ctx, customer.ID)
}

// EnableAutopay enables autopay for a customer
func (cr *CustomerRepository) EnableAutopay(ctx context.Context, customerID int64) error {
	return cr.UpdateFields(ctx, customerID, map[string]interface{}{
		"autopay_enabled":         true,
		"autopay_failed_attempts": 0,
	})
}

// EnableAutopayByTelegramID enables autopay by telegram ID
func (cr *CustomerRepository) EnableAutopayByTelegramID(ctx context.Context, telegramID int64) error {
	customer, err := cr.FindByTelegramId(ctx, telegramID)
	if err != nil {
		return err
	}
	if customer == nil {
		return nil
	}
	return cr.EnableAutopay(ctx, customer.ID)
}
