package database

import (
	"context"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(poll *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: poll}
}

type Customer struct {
	ID               int64      `db:"id"`
	TelegramID       int64      `db:"telegram_id"`
	ExpireAt         *time.Time `db:"expire_at"`
	CreatedAt        time.Time  `db:"created_at"`
	SubscriptionLink *string    `db:"subscription_link"`
	Language         string     `db:"language"`
}

func (cr *CustomerRepository) FindByExpirationRange(ctx context.Context, startDate, endDate time.Time) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language").
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
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language").
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
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language").
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
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return &customer, nil
}

func (cr *CustomerRepository) Create(ctx context.Context, customer *Customer) error {
	buildInsert := sq.Insert("customer").
		Columns("telegram_id", "expire_at", "language").
		PlaceholderFormat(sq.Dollar).
		Values(customer.TelegramID, customer.ExpireAt, customer.Language)

	sql, arg, err := buildInsert.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = cr.pool.Exec(ctx, sql, arg...)
	if err != nil {
		return fmt.Errorf("failed to insert customer: %w", err)
	}
	return nil
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

	result, err := cr.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no customer found with id: %d", id)
	}

	return nil
}
