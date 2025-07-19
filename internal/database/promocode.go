package database

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

type Promocode struct {
	ID        int64     `db:"id"`
	Code      string    `db:"code"`
	Months    int       `db:"months"`
	UsesLeft  int       `db:"uses_left"`
	CreatedBy int64     `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

type PromocodeRepository struct {
	pool *pgxpool.Pool
}

func NewPromocodeRepository(pool *pgxpool.Pool) *PromocodeRepository {
	return &PromocodeRepository{pool: pool}
}

func (r *PromocodeRepository) Create(ctx context.Context, promo *Promocode) (*Promocode, error) {
	sql, args, err := sq.Insert("promocode").
		Columns("code", "months", "uses_left", "created_by").
		Values(promo.Code, promo.Months, promo.UsesLeft, promo.CreatedBy).
		Suffix("RETURNING id, created_at").
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert promocode: %w", err)
	}
	row := r.pool.QueryRow(ctx, sql, args...)
	if err := row.Scan(&promo.ID, &promo.CreatedAt); err != nil {
		return nil, err
	}
	return promo, nil
}
