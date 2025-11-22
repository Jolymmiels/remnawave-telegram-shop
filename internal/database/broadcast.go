package database

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type BroadcastRepository struct {
	pool *pgxpool.Pool
}

const (
	BroadcastAll      = "all"
	BroadcastActive   = "active"
	BroadcastInactive = "inactive"
)

type Broadcast struct {
	ID        int64     `db:"id"`
	Content   string    `db:"content"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
	Language  string    `db:"language"`
}

type BroadcastListParams struct {
	Type     string // "", "all", "active", "inactive" ("" == нет фильтра)
	Language string // "" == нет фильтра
	Limit    int
	Offset   int
	SortBy   string // только "created_at"
	Desc     bool
}

func NewBroadcastRepository(pool *pgxpool.Pool) *BroadcastRepository {
	return &BroadcastRepository{pool: pool}
}

func (r *BroadcastRepository) GetByID(ctx context.Context, id int64) (*Broadcast, error) {
	b := sq.Select("id", "content", "type", "created_at", "language").
		From("broadcast").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select by id: %w", err)
	}
	row := r.pool.QueryRow(ctx, sql, args...)

	var br Broadcast
	if err := row.Scan(&br.ID, &br.Content, &br.Type, &br.CreatedAt, &br.Language); err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scan broadcast by id: %w", err)
	}
	return &br, nil
}

func (r *BroadcastRepository) List(ctx context.Context, p BroadcastListParams) (*[]Broadcast, error) {
	b := sq.Select("id", "content", "type", "created_at", "language").
		From("broadcast").
		PlaceholderFormat(sq.Dollar)

	if p.Type != "" && p.Type != BroadcastAll {
		b = b.Where(sq.Eq{"type": p.Type})
	}
	if p.Language != "" {
		b = b.Where(sq.Eq{"language": p.Language})
	}

	order := "created_at DESC"
	if p.SortBy == "created_at" && !p.Desc {
		order = "created_at ASC"
	}
	b = b.OrderBy(order)

	if p.Limit > 0 {
		b = b.Limit(uint64(p.Limit))
	}
	if p.Offset > 0 {
		b = b.Offset(uint64(p.Offset))
	}

	sql, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query list: %w", err)
	}
	defer rows.Close()

	var out []Broadcast
	for rows.Next() {
		var br Broadcast
		if err := rows.Scan(&br.ID, &br.Content, &br.Type, &br.CreatedAt, &br.Language); err != nil {
			return nil, err
		}
		out = append(out, br)
	}
	return &out, nil
}

func (r *BroadcastRepository) CreateBroadcast(ctx context.Context, broadcast *Broadcast) (*Broadcast, error) {
	query := sq.Insert("broadcast").
		Columns("content", "type", "language").
		Values(broadcast.Content, broadcast.Type, broadcast.Language).
		PlaceholderFormat(sq.Dollar).
		Suffix("RETURNING id, created_at")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	row := r.pool.QueryRow(ctx, sql, args...)
	var id int64
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to insert customer: %w", err)
	}
	broadcast.ID = id
	broadcast.CreatedAt = createdAt

	return broadcast, nil
}
