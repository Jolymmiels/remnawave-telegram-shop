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
