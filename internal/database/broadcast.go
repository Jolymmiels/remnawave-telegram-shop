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
	BroadcastAll            = "all"
	BroadcastActive         = "active"
	BroadcastInactive       = "inactive"
	BroadcastNoSubscription = "no_subscription"

	BroadcastStatusPending    = "pending"
	BroadcastStatusInProgress = "in_progress"
	BroadcastStatusCompleted  = "completed"
	BroadcastStatusFailed     = "failed"
)

type Broadcast struct {
	ID           int64     `db:"id" json:"ID"`
	Content      string    `db:"content" json:"Content"`
	Type         string    `db:"type" json:"Type"`
	CreatedAt    time.Time `db:"created_at" json:"CreatedAt"`
	Language     string    `db:"language" json:"Language"`
	Status       string    `db:"status" json:"Status"`
	TotalCount   int       `db:"total_count" json:"TotalCount"`
	SentCount    int       `db:"sent_count" json:"SentCount"`
	FailedCount  int       `db:"failed_count" json:"FailedCount"`
	BlockedCount int       `db:"blocked_count" json:"BlockedCount"`
}

type BroadcastListParams struct {
	Type     string // "", "all", "active", "inactive" ("" == нет фильтра)
	Language string // "" == нет фильтра
	Status   string // "", "pending", "in_progress", "completed", "failed"
	Limit    int
	Offset   int
	SortBy   string // только "created_at"
	Desc     bool
}

func NewBroadcastRepository(pool *pgxpool.Pool) *BroadcastRepository {
	return &BroadcastRepository{pool: pool}
}

func (r *BroadcastRepository) GetByID(ctx context.Context, id int64) (*Broadcast, error) {
	b := sq.Select("id", "content", "type", "created_at", "language", "status", "total_count", "sent_count", "failed_count", "blocked_count").
		From("broadcast").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select by id: %w", err)
	}
	row := r.pool.QueryRow(ctx, sql, args...)

	var br Broadcast
	if err := row.Scan(&br.ID, &br.Content, &br.Type, &br.CreatedAt, &br.Language, &br.Status, &br.TotalCount, &br.SentCount, &br.FailedCount, &br.BlockedCount); err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scan broadcast by id: %w", err)
	}
	return &br, nil
}

func (r *BroadcastRepository) List(ctx context.Context, p BroadcastListParams) (*[]Broadcast, error) {
	b := sq.Select("id", "content", "type", "created_at", "language", "status", "total_count", "sent_count", "failed_count", "blocked_count").
		From("broadcast").
		PlaceholderFormat(sq.Dollar)

	if p.Type != "" && p.Type != BroadcastAll {
		b = b.Where(sq.Eq{"type": p.Type})
	}
	if p.Language != "" {
		b = b.Where(sq.Eq{"language": p.Language})
	}
	if p.Status != "" {
		b = b.Where(sq.Eq{"status": p.Status})
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
		if err := rows.Scan(&br.ID, &br.Content, &br.Type, &br.CreatedAt, &br.Language, &br.Status, &br.TotalCount, &br.SentCount, &br.FailedCount, &br.BlockedCount); err != nil {
			return nil, err
		}
		out = append(out, br)
	}
	return &out, nil
}

func (r *BroadcastRepository) CreateBroadcast(ctx context.Context, broadcast *Broadcast) (*Broadcast, error) {
	query := sq.Insert("broadcast").
		Columns("content", "type", "language", "status", "total_count", "sent_count", "failed_count", "blocked_count").
		Values(broadcast.Content, broadcast.Type, broadcast.Language, BroadcastStatusPending, 0, 0, 0, 0).
		PlaceholderFormat(sq.Dollar).
		Suffix("RETURNING id, created_at, status")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	row := r.pool.QueryRow(ctx, sql, args...)
	var id int64
	var createdAt time.Time
	var status string
	if err := row.Scan(&id, &createdAt, &status); err != nil {
		return nil, fmt.Errorf("failed to insert broadcast: %w", err)
	}
	broadcast.ID = id
	broadcast.CreatedAt = createdAt
	broadcast.Status = status
	broadcast.TotalCount = 0
	broadcast.SentCount = 0
	broadcast.FailedCount = 0
	broadcast.BlockedCount = 0

	return broadcast, nil
}

func (r *BroadcastRepository) Delete(ctx context.Context, id int64) error {
	query := sq.Delete("broadcast").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to delete broadcast: %w", err)
	}

	return nil
}

func (r *BroadcastRepository) UpdateBroadcastStats(ctx context.Context, id int64, status string, total, sent, failed, blocked int) error {
	query := sq.Update("broadcast").
		Set("status", status).
		Set("total_count", total).
		Set("sent_count", sent).
		Set("failed_count", failed).
		Set("blocked_count", blocked).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update broadcast stats: %w", err)
	}

	return nil
}
