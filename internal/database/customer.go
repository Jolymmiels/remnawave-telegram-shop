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
	ID                   int64      `db:"id"`
	TelegramID           int64      `db:"telegram_id"`
	ExpireAt             *time.Time `db:"expire_at"`
	CreatedAt            time.Time  `db:"created_at"`
	SubscriptionLink     *string    `db:"subscription_link"`
	Language             string     `db:"language"`
	Login                *string    `db:"login"`
	PasswordHash         *string    `db:"password_hash"`
	AuthType             string     `db:"auth_type"`
	RemnawaveUserUUID    *string    `db:"remnawave_user_uuid"`
	IsActive             bool       `db:"is_active"`
	LastLoginAt          *time.Time `db:"last_login_at"`
	MergedIntoCustomerID *int64     `db:"merged_into_customer_id"`
}

type customerScanner interface {
	Scan(dest ...interface{}) error
}

func scanCustomer(scanner customerScanner, customer *Customer) error {
	var telegramID *int64

	if err := scanner.Scan(
		&customer.ID,
		&telegramID,
		&customer.ExpireAt,
		&customer.CreatedAt,
		&customer.SubscriptionLink,
		&customer.Language,
		&customer.Login,
		&customer.PasswordHash,
		&customer.AuthType,
		&customer.RemnawaveUserUUID,
		&customer.IsActive,
		&customer.LastLoginAt,
		&customer.MergedIntoCustomerID,
	); err != nil {
		return err
	}

	if telegramID != nil {
		customer.TelegramID = *telegramID
	} else {
		customer.TelegramID = 0
	}

	return nil
}

const customerSelectColumns = "id, telegram_id, expire_at, created_at, subscription_link, language, login, password_hash, auth_type, remnawave_user_uuid, is_active, last_login_at, merged_into_customer_id"

func (cr *CustomerRepository) FindByExpirationRange(ctx context.Context, startDate, endDate time.Time) (*[]Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "login", "password_hash", "auth_type", "remnawave_user_uuid", "is_active", "last_login_at", "merged_into_customer_id").
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
		err := scanCustomer(rows, &customer)
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
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "login", "password_hash", "auth_type", "remnawave_user_uuid", "is_active", "last_login_at", "merged_into_customer_id").
		From("customer").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var customer Customer

	err = scanCustomer(cr.pool.QueryRow(ctx, sql, args...), &customer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return &customer, nil
}

func (cr *CustomerRepository) FindByTelegramId(ctx context.Context, telegramId int64) (*Customer, error) {
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "login", "password_hash", "auth_type", "remnawave_user_uuid", "is_active", "last_login_at", "merged_into_customer_id").
		From("customer").
		Where(sq.Eq{"telegram_id": telegramId}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := buildSelect.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var customer Customer

	err = scanCustomer(cr.pool.QueryRow(ctx, sql, args...), &customer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}
	return &customer, nil
}

func (cr *CustomerRepository) Create(ctx context.Context, customer *Customer) (*Customer, error) {
	return cr.FindOrCreate(ctx, customer)
}

func (cr *CustomerRepository) FindOrCreate(ctx context.Context, customer *Customer) (*Customer, error) {
	query := `
		INSERT INTO customer (telegram_id, expire_at, language)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_id) DO UPDATE SET telegram_id = customer.telegram_id
		RETURNING id, telegram_id, expire_at, created_at, subscription_link, language, login, password_hash, auth_type, remnawave_user_uuid, is_active, last_login_at, merged_into_customer_id
	`

	row := cr.pool.QueryRow(ctx, query, customer.TelegramID, customer.ExpireAt, customer.Language)
	var result Customer
	if err := scanCustomer(row, &result); err != nil {
		return nil, fmt.Errorf("failed to find or create customer: %w", err)
	}

	slog.Info("user found or created in bot database", "telegramId", utils.MaskHalfInt64(result.TelegramID))
	return &result, nil
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
	buildSelect := sq.Select("id", "telegram_id", "expire_at", "created_at", "subscription_link", "language", "login", "password_hash", "auth_type", "remnawave_user_uuid", "is_active", "last_login_at", "merged_into_customer_id").
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
		err := scanCustomer(rows, &customer)
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

func (cr *CustomerRepository) FindAll(ctx context.Context) (*[]Customer, error) {
	query := `
		SELECT id, telegram_id, expire_at, created_at, subscription_link, language, login, password_hash, auth_type, remnawave_user_uuid, is_active, last_login_at, merged_into_customer_id
		FROM customer
		ORDER BY created_at DESC
	`

	rows, err := cr.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var customer Customer
		err := scanCustomer(rows, &customer)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &customers, nil
}

func (cr *CustomerRepository) FindByLogin(ctx context.Context, login string) (*Customer, error) {
	query := fmt.Sprintf("SELECT %s FROM customer WHERE login = $1", customerSelectColumns)

	var customer Customer
	err := scanCustomer(cr.pool.QueryRow(ctx, query, login), &customer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query customer by login: %w", err)
	}

	return &customer, nil
}

func (cr *CustomerRepository) CreateWebCustomer(ctx context.Context, customer *Customer) (*Customer, error) {
	query := `
		INSERT INTO customer (login, password_hash, language, auth_type, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, telegram_id, expire_at, created_at, subscription_link, language, login, password_hash, auth_type, remnawave_user_uuid, is_active, last_login_at, merged_into_customer_id
	`

	var result Customer
	err := scanCustomer(cr.pool.QueryRow(
		ctx,
		query,
		customer.Login,
		customer.PasswordHash,
		customer.Language,
		customer.AuthType,
		customer.IsActive,
	), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create web customer: %w", err)
	}

	return &result, nil
}
