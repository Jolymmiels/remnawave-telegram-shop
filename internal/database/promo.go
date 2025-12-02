package database

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Promo struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	BonusDays int        `json:"bonus_days"` // Additional days to add to subscription
	MaxUses   *int       `json:"max_uses"`   // NULL for unlimited uses
	UsedCount int        `json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at"` // NULL for no expiration
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
}

type PromoUsage struct {
	ID         int64     `json:"id"`
	PromoID    int64     `json:"promo_id"`
	CustomerID int64     `json:"customer_id"`
	UsedAt     time.Time `json:"used_at"`
}

type CreatePromoRequest struct {
	Code      string     `json:"code"`
	BonusDays int        `json:"bonus_days"`
	MaxUses   *int       `json:"max_uses"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type ValidatePromoResponse struct {
	Valid     bool   `json:"valid"`
	BonusDays int    `json:"bonus_days"`
	Message   string `json:"message"`
	PromoID   int64  `json:"promo_id,omitempty"`
}

type PromoRepository struct {
	db *pgxpool.Pool
}

func NewPromoRepository(db *pgxpool.Pool) *PromoRepository {
	return &PromoRepository{db: db}
}

func (r *PromoRepository) Create(ctx context.Context, req *CreatePromoRequest) (*Promo, error) {
	query := squirrel.Insert("promo").
		Columns("code", "bonus_days", "max_uses", "expires_at").
		Values(req.Code, req.BonusDays, req.MaxUses, req.ExpiresAt).
		Suffix("RETURNING id, code, bonus_days, max_uses, used_count, expires_at, active, created_at").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var p Promo
	row := r.db.QueryRow(ctx, sql, args...)
	err = row.Scan(&p.ID, &p.Code, &p.BonusDays, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PromoRepository) GetByID(ctx context.Context, id int64) (*Promo, error) {
	query := squirrel.Select("id", "code", "bonus_days", "max_uses", "used_count", "expires_at", "active", "created_at").
		From("promo").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var p Promo
	row := r.db.QueryRow(ctx, sql, args...)
	err = row.Scan(&p.ID, &p.Code, &p.BonusDays, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (r *PromoRepository) GetByCode(ctx context.Context, code string) (*Promo, error) {
	query := squirrel.Select("id", "code", "bonus_days", "max_uses", "used_count", "expires_at", "active", "created_at").
		From("promo").
		Where(squirrel.Eq{"code": code}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var p Promo
	row := r.db.QueryRow(ctx, sql, args...)
	err = row.Scan(&p.ID, &p.Code, &p.BonusDays, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (r *PromoRepository) GetAll(ctx context.Context) ([]*Promo, error) {
	query := squirrel.Select("id", "code", "bonus_days", "max_uses", "used_count", "expires_at", "active", "created_at").
		From("promo").
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promos []*Promo
	for rows.Next() {
		var p Promo
		err := rows.Scan(&p.ID, &p.Code, &p.BonusDays, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		promos = append(promos, &p)
	}

	return promos, nil
}

func (r *PromoRepository) Update(ctx context.Context, id int64, active bool) error {
	query := squirrel.Update("promo").
		Set("active", active).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *PromoRepository) Delete(ctx context.Context, id int64) error {
	query := squirrel.Delete("promo").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *PromoRepository) HasCustomerUsedPromo(ctx context.Context, promoID, customerID int64) (bool, error) {
	query := squirrel.Select("1").
		From("promo_usage").
		Where(squirrel.Eq{"promo_id": promoID, "customer_id": customerID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return false, err
	}

	var exists int
	row := r.db.QueryRow(ctx, sql, args...)
	err = row.Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *PromoRepository) RecordPromoUsage(ctx context.Context, promoID, customerID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	insertQuery := squirrel.Insert("promo_usage").
		Columns("promo_id", "customer_id").
		Values(promoID, customerID).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := insertQuery.ToSql()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	// Increment used_count
	updateQuery := squirrel.Update("promo").
		Set("used_count", squirrel.Expr("used_count + 1")).
		Where(squirrel.Eq{"id": promoID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err = updateQuery.ToSql()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// PromoUsageWithCustomer contains usage info with customer telegram_id
type PromoUsageWithCustomer struct {
	ID          int64     `json:"id"`
	PromoID     int64     `json:"promo_id"`
	TelegramID  int64     `json:"telegram_id"`
	TgUsername  *string   `json:"tg_username"`
	TgFirstName *string   `json:"tg_first_name"`
	TgLastName  *string   `json:"tg_last_name"`
	UsedAt      time.Time `json:"used_at"`
}

// GetPromoUsages returns all usages for a promo with customer telegram IDs
func (r *PromoRepository) GetPromoUsages(ctx context.Context, promoID int64) ([]PromoUsageWithCustomer, error) {
	query := `
		SELECT pu.id, pu.promo_id, c.telegram_id, c.tg_username, c.tg_first_name, c.tg_last_name, pu.used_at
		FROM promo_usage pu
		JOIN customer c ON c.id = pu.customer_id
		WHERE pu.promo_id = $1
		ORDER BY pu.used_at DESC
	`

	rows, err := r.db.Query(ctx, query, promoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []PromoUsageWithCustomer
	for rows.Next() {
		var u PromoUsageWithCustomer
		if err := rows.Scan(&u.ID, &u.PromoID, &u.TelegramID, &u.TgUsername, &u.TgFirstName, &u.TgLastName, &u.UsedAt); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}

	return usages, rows.Err()
}

// CustomerPromoUsage contains promo info used by customer
type CustomerPromoUsage struct {
	PromoID   int64     `json:"promo_id"`
	Code      string    `json:"code"`
	BonusDays int       `json:"bonus_days"`
	UsedAt    time.Time `json:"used_at"`
}

// GetCustomerPromoUsages returns all promos used by a customer
func (r *PromoRepository) GetCustomerPromoUsages(ctx context.Context, telegramID int64) ([]CustomerPromoUsage, error) {
	query := `
		SELECT p.id, p.code, p.bonus_days, pu.used_at
		FROM promo_usage pu
		JOIN promo p ON p.id = pu.promo_id
		JOIN customer c ON c.id = pu.customer_id
		WHERE c.telegram_id = $1
		ORDER BY pu.used_at DESC
	`

	rows, err := r.db.Query(ctx, query, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []CustomerPromoUsage
	for rows.Next() {
		var u CustomerPromoUsage
		if err := rows.Scan(&u.PromoID, &u.Code, &u.BonusDays, &u.UsedAt); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}

	return usages, rows.Err()
}
