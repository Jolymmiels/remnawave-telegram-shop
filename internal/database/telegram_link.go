package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrTelegramLinkInvalid        = errors.New("telegram link token is invalid")
	ErrTelegramLinkExpired        = errors.New("telegram link token has expired")
	ErrTelegramLinkUsed           = errors.New("telegram link token has already been used")
	ErrTelegramAlreadyLinked      = errors.New("telegram account is already linked to another customer")
	ErrCustomerAlreadyLinked      = errors.New("customer already linked to another telegram account")
	ErrTelegramLinkCustomerAbsent = errors.New("customer for telegram link token not found")
	ErrTelegramMergeNotAllowed    = errors.New("telegram account cannot be merged into current customer")
)

type TelegramLinkRequest struct {
	ID         int64
	CustomerID int64
	Token      string
	TelegramID *int64
	CreatedAt  time.Time
	ExpiresAt  time.Time
	UsedAt     *time.Time
}

type TelegramLinkRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramLinkRepository(pool *pgxpool.Pool) *TelegramLinkRepository {
	return &TelegramLinkRepository{pool: pool}
}

func (r *TelegramLinkRepository) Create(ctx context.Context, customerID int64, token string, expiresAt time.Time) (*TelegramLinkRequest, error) {
	query := `
		INSERT INTO telegram_link_request (customer_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, customer_id, token, telegram_id, created_at, expires_at, used_at
	`

	var req TelegramLinkRequest
	var telegramID *int64
	if err := r.pool.QueryRow(ctx, query, customerID, token, expiresAt).Scan(
		&req.ID,
		&req.CustomerID,
		&req.Token,
		&telegramID,
		&req.CreatedAt,
		&req.ExpiresAt,
		&req.UsedAt,
	); err != nil {
		return nil, fmt.Errorf("create telegram link request: %w", err)
	}

	req.TelegramID = telegramID
	return &req, nil
}

func (r *TelegramLinkRepository) Consume(ctx context.Context, token string, telegramID int64, language string) (*Customer, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin telegram link transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	linkRequest, err := r.findRequestForUpdate(ctx, tx, token)
	if err != nil {
		return nil, err
	}

	if linkRequest.UsedAt != nil {
		return nil, ErrTelegramLinkUsed
	}
	if time.Now().UTC().After(linkRequest.ExpiresAt) {
		return nil, ErrTelegramLinkExpired
	}

	customer, err := r.findCustomerForUpdate(ctx, tx, linkRequest.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrTelegramLinkCustomerAbsent
	}

	if customer.TelegramID != 0 && customer.TelegramID != telegramID {
		return nil, ErrCustomerAlreadyLinked
	}

	existingCustomer, err := r.findCustomerByTelegramIDForUpdate(ctx, tx, telegramID)
	if err != nil {
		return nil, err
	}
	if existingCustomer != nil && existingCustomer.ID != customer.ID {
		updatedCustomer, err := r.mergeWebCustomerIntoTelegramCustomer(ctx, tx, customer, existingCustomer, linkRequest, language)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit telegram link merge transaction: %w", err)
		}

		return updatedCustomer, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE customer
		SET telegram_id = $1,
		    language = CASE WHEN $2 <> '' THEN $2 ELSE language END
		WHERE id = $3
	`, telegramID, language, customer.ID); err != nil {
		return nil, fmt.Errorf("update customer telegram link: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE telegram_link_request
		SET telegram_id = $1,
		    used_at = NOW()
		WHERE id = $2
	`, telegramID, linkRequest.ID); err != nil {
		return nil, fmt.Errorf("mark telegram link request used: %w", err)
	}

	updatedCustomer, err := r.findCustomerForUpdate(ctx, tx, customer.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit telegram link transaction: %w", err)
	}

	return updatedCustomer, nil
}

func (r *TelegramLinkRepository) mergeWebCustomerIntoTelegramCustomer(
	ctx context.Context,
	tx pgx.Tx,
	webCustomer *Customer,
	telegramCustomer *Customer,
	linkRequest *TelegramLinkRequest,
	language string,
) (*Customer, error) {
	if !isEmptyWebCustomer(webCustomer) {
		return nil, ErrTelegramAlreadyLinked
	}
	if telegramCustomer.Login != nil || telegramCustomer.PasswordHash != nil {
		return nil, ErrTelegramMergeNotAllowed
	}
	if webCustomer.Login == nil || webCustomer.PasswordHash == nil {
		return nil, ErrTelegramMergeNotAllowed
	}

	hasPurchases, err := r.customerHasPurchases(ctx, tx, webCustomer.ID)
	if err != nil {
		return nil, err
	}
	if hasPurchases {
		return nil, ErrTelegramMergeNotAllowed
	}

	mergedLanguage := telegramCustomer.Language
	if language != "" {
		mergedLanguage = language
	} else if webCustomer.Language != "" {
		mergedLanguage = webCustomer.Language
	}

	if _, err := tx.Exec(ctx, `
		UPDATE customer
		SET login = NULL,
		    password_hash = NULL,
		    is_active = FALSE,
		    merged_into_customer_id = $1
		WHERE id = $2
	`, telegramCustomer.ID, webCustomer.ID); err != nil {
		return nil, fmt.Errorf("mark web customer merged: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE customer
		SET login = $1,
		    password_hash = $2,
		    auth_type = $3,
		    is_active = TRUE,
		    language = $4,
		    last_login_at = COALESCE(last_login_at, $5)
		WHERE id = $6
	`, webCustomer.Login, webCustomer.PasswordHash, "web", mergedLanguage, webCustomer.LastLoginAt, telegramCustomer.ID); err != nil {
		return nil, fmt.Errorf("update telegram customer during merge: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE telegram_link_request
		SET telegram_id = $1,
		    used_at = NOW()
		WHERE id = $2
	`, telegramCustomer.TelegramID, linkRequest.ID); err != nil {
		return nil, fmt.Errorf("mark telegram link request used after merge: %w", err)
	}

	updatedCustomer, err := r.findCustomerForUpdate(ctx, tx, telegramCustomer.ID)
	if err != nil {
		return nil, err
	}

	return updatedCustomer, nil
}

func (r *TelegramLinkRepository) findRequestForUpdate(ctx context.Context, tx pgx.Tx, token string) (*TelegramLinkRequest, error) {
	query := `
		SELECT id, customer_id, token, telegram_id, created_at, expires_at, used_at
		FROM telegram_link_request
		WHERE token = $1
		FOR UPDATE
	`

	var req TelegramLinkRequest
	var telegramID *int64
	err := tx.QueryRow(ctx, query, token).Scan(
		&req.ID,
		&req.CustomerID,
		&req.Token,
		&telegramID,
		&req.CreatedAt,
		&req.ExpiresAt,
		&req.UsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTelegramLinkInvalid
		}
		return nil, fmt.Errorf("query telegram link request: %w", err)
	}

	req.TelegramID = telegramID
	return &req, nil
}

func (r *TelegramLinkRepository) findCustomerForUpdate(ctx context.Context, tx pgx.Tx, customerID int64) (*Customer, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM customer
		WHERE id = $1
		FOR UPDATE
	`, customerSelectColumns)

	var customer Customer
	if err := scanCustomer(tx.QueryRow(ctx, query, customerID), &customer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query customer for telegram link: %w", err)
	}

	return &customer, nil
}

func (r *TelegramLinkRepository) findCustomerByTelegramIDForUpdate(ctx context.Context, tx pgx.Tx, telegramID int64) (*Customer, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM customer
		WHERE telegram_id = $1
		FOR UPDATE
	`, customerSelectColumns)

	var customer Customer
	if err := scanCustomer(tx.QueryRow(ctx, query, telegramID), &customer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query customer by telegram id for telegram link: %w", err)
	}

	return &customer, nil
}

func (r *TelegramLinkRepository) customerHasPurchases(ctx context.Context, tx pgx.Tx, customerID int64) (bool, error) {
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM purchase
		WHERE customer_id = $1
	`, customerID).Scan(&count); err != nil {
		return false, fmt.Errorf("count customer purchases: %w", err)
	}

	return count > 0, nil
}

func isEmptyWebCustomer(customer *Customer) bool {
	if customer == nil {
		return false
	}

	return customer.TelegramID == 0 &&
		customer.SubscriptionLink == nil &&
		customer.ExpireAt == nil &&
		customer.RemnawaveUserUUID == nil &&
		customer.MergedIntoCustomerID == nil
}
