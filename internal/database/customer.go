package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"remnawave-tg-shop-bot/utils"
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
	ID               int64      `db:"id" json:"id"`
	TelegramID       int64      `db:"telegram_id" json:"telegram_id"`
	ExpireAt         *time.Time `db:"expire_at" json:"expire_at"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	SubscriptionLink *string    `db:"subscription_link" json:"subscription_link"`
	Language         string     `db:"language" json:"language"`
	IsBlocked        bool       `db:"is_blocked" json:"is_blocked"`
}

func (cr *CustomerRepository) FindByExpirationRange(ctx context.Context, startDate, endDate time.Time) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
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
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
						&customer.Language,
			&customer.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindById(ctx context.Context, id int64) (*Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
		From("customer").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var customer Customer

	err = cr.pool.QueryRow(ctx, sql, args...).Scan(
		&customer.ID,
		&customer.TelegramID,
		&customer.ExpireAt,
		&customer.CreatedAt,
		&customer.SubscriptionLink,
					&customer.Language,
			&customer.IsBlocked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return &customer, nil
}

func (cr *CustomerRepository) FindByTelegramId(ctx context.Context, telegramId int64) (*Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
		From("customer").
		Where(sq.Eq{"telegram_id": telegramId}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var customer Customer

	err = cr.pool.QueryRow(ctx, sql, args...).Scan(
		&customer.ID,
		&customer.TelegramID,
		&customer.ExpireAt,
		&customer.CreatedAt,
		&customer.SubscriptionLink,
					&customer.Language,
			&customer.IsBlocked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return &customer, nil
}

func (cr *CustomerRepository) Create(ctx context.Context, customer *Customer) (*Customer, error) {
	buildInsert := sq.Insert("customer").
		Columns("telegram_id", "expire_at", "language").
		PlaceholderFormat(sq.Dollar).
		Values(customer.TelegramID, customer.ExpireAt, customer.Language).
		Suffix("RETURNING id, created_at")
	sqlStr, args, err := buildInsert.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	row := cr.pool.QueryRow(ctx, sqlStr, args...)
	var id int64
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to insert customer: %w", err)
	}
	customer.ID = id
	customer.CreatedAt = createdAt

	slog.Info("user created in bot database", "telegramId", utils.MaskHalfInt64(customer.TelegramID))
	return customer, nil
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
		return fmt.Errorf("no customer found with id: %d", utils.MaskHalfInt64(id))
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (cr *CustomerRepository) FindByTelegramIds(ctx context.Context, telegramIDs []int64) ([]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
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
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
						&customer.Language,
			&customer.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
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

func (cr *CustomerRepository) FindAll(ctx context.Context) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
		From("customer").
		PlaceholderFormat(sq.Dollar)

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
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
						&customer.Language,
			&customer.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindNonExpired(ctx context.Context) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
		From("customer").
		Where(sq.Gt{"expire_at": time.Now()}).
		PlaceholderFormat(sq.Dollar)

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
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
						&customer.Language,
			&customer.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindExpired(ctx context.Context) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "is_blocked").
		From("customer").
		Where(sq.LtOrEq{"expire_at": time.Now()}).
		PlaceholderFormat(sq.Dollar)

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
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
						&customer.Language,
			&customer.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over customer rows: %w", err)
	}

	return &customers, nil
}
