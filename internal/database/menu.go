package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"remnawave-tg-shop-bot/internal/menu"
)

type MenuRepository struct {
	pool *pgxpool.Pool
}

var _ menu.Store = (*MenuRepository)(nil)

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

func (r *MenuRepository) LoadPhoto(ctx context.Context, botID int64, path string) (menu.PhotoRecord, error) {
	var photo menu.PhotoRecord
	err := r.pool.QueryRow(ctx, "SELECT content_hash, file_id FROM bot_menu_photo WHERE bot_id = $1 AND path = $2", botID, path).Scan(&photo.Hash, &photo.FileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return photo, nil
	}
	return photo, err
}

func (r *MenuRepository) SavePhoto(ctx context.Context, botID int64, path string, photo menu.PhotoRecord) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO bot_menu_photo (bot_id, path, content_hash, file_id) VALUES ($1, $2, $3, $4)
		ON CONFLICT (bot_id, path) DO UPDATE SET content_hash = EXCLUDED.content_hash, file_id = EXCLUDED.file_id`,
		botID, path, photo.Hash, photo.FileID)
	return err
}

func (r *MenuRepository) ListPins(ctx context.Context, botID, chatID int64) ([]int, error) {
	rows, err := r.pool.Query(ctx, "SELECT message_id FROM bot_menu_pin WHERE bot_id = $1 AND chat_id = $2 ORDER BY message_id", botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *MenuRepository) TrackPin(ctx context.Context, botID, chatID int64, messageID int) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO bot_menu_pin (bot_id, chat_id, message_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", botID, chatID, messageID)
	return err
}

func (r *MenuRepository) ForgetPin(ctx context.Context, botID, chatID int64, messageID int) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM bot_menu_pin WHERE bot_id = $1 AND chat_id = $2 AND message_id = $3", botID, chatID, messageID)
	return err
}
