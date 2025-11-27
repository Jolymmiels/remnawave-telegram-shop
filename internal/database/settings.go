package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Setting struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SettingsRepository struct {
	pool  *pgxpool.Pool
	cache map[string]string
	mu    sync.RWMutex
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{
		pool:  pool,
		cache: make(map[string]string),
	}
}

// LoadAll loads all settings from database into cache
func (sr *SettingsRepository) LoadAll(ctx context.Context) error {
	query := sq.Select("key", "value").
		From("settings").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := sr.pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	sr.mu.Lock()
	defer sr.mu.Unlock()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("failed to scan setting: %w", err)
		}
		sr.cache[key] = value
	}

	return rows.Err()
}

// Get returns a setting value from cache
func (sr *SettingsRepository) Get(key string) string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.cache[key]
}

// GetInt returns a setting value as int
func (sr *SettingsRepository) GetInt(key string, defaultValue int) int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	if val, ok := sr.cache[key]; ok && val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// GetBool returns a setting value as bool
func (sr *SettingsRepository) GetBool(key string) bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.cache[key] == "true"
}

// GetAll returns all settings as a map
func (sr *SettingsRepository) GetAll() map[string]string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	result := make(map[string]string, len(sr.cache))
	for k, v := range sr.cache {
		result[k] = v
	}
	return result
}

// Set updates a single setting in database and cache
func (sr *SettingsRepository) Set(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`

	_, err := sr.pool.Exec(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to update setting %s: %w", key, err)
	}

	sr.mu.Lock()
	sr.cache[key] = value
	sr.mu.Unlock()

	return nil
}

// SetMultiple updates multiple settings in a single transaction
func (sr *SettingsRepository) SetMultiple(ctx context.Context, settings map[string]string) error {
	tx, err := sr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`

	for key, value := range settings {
		if _, err := tx.Exec(ctx, query, key, value); err != nil {
			return fmt.Errorf("failed to update setting %s: %w", key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	sr.mu.Lock()
	for key, value := range settings {
		sr.cache[key] = value
	}
	sr.mu.Unlock()

	return nil
}

// GetAllFromDB returns all settings directly from database
func (sr *SettingsRepository) GetAllFromDB(ctx context.Context) ([]Setting, error) {
	query := sq.Select("key", "value", "updated_at").
		From("settings").
		OrderBy("key").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := sr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, s)
	}

	return settings, rows.Err()
}
