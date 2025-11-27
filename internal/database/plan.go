package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Plan struct {
	ID           int64      `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Price1       int        `db:"price_1" json:"price_1"`
	Price3       int        `db:"price_3" json:"price_3"`
	Price6       int        `db:"price_6" json:"price_6"`
	Price12      int        `db:"price_12" json:"price_12"`
	TrafficLimit int        `db:"traffic_limit" json:"traffic_limit"`
	DeviceLimit  *int       `db:"device_limit" json:"device_limit"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	IsDefault    bool       `db:"is_default" json:"is_default"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// GetPrice returns the price for a given month period
func (p *Plan) GetPrice(month int) int {
	switch month {
	case 1:
		return p.Price1
	case 3:
		return p.Price3
	case 6:
		return p.Price6
	case 12:
		return p.Price12
	default:
		return p.Price1
	}
}

// GetStarsPrice calculates stars price based on exchange rate
func (p *Plan) GetStarsPrice(month int, exchangeRate float64) int {
	price := p.GetPrice(month)
	return int(math.Round(float64(price) * exchangeRate))
}

type PlanRepository struct {
	pool *pgxpool.Pool
}

func NewPlanRepository(pool *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{pool: pool}
}

var planColumns = []string{
	"id", "name", "price_1", "price_3", "price_6", "price_12",
	"traffic_limit", "device_limit", "is_active", "is_default",
	"created_at", "updated_at",
}

func (pr *PlanRepository) scanPlan(row pgx.Row) (*Plan, error) {
	var plan Plan
	err := row.Scan(
		&plan.ID, &plan.Name, &plan.Price1, &plan.Price3, &plan.Price6, &plan.Price12,
		&plan.TrafficLimit, &plan.DeviceLimit, &plan.IsActive, &plan.IsDefault,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (pr *PlanRepository) scanPlans(rows pgx.Rows) ([]Plan, error) {
	var plans []Plan
	for rows.Next() {
		var plan Plan
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Price1, &plan.Price3, &plan.Price6, &plan.Price12,
			&plan.TrafficLimit, &plan.DeviceLimit, &plan.IsActive, &plan.IsDefault,
			&plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

// FindAll returns all plans
func (pr *PlanRepository) FindAll(ctx context.Context) ([]Plan, error) {
	query := sq.Select(planColumns...).
		From("plan").
		OrderBy("is_default DESC", "name ASC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query plans: %w", err)
	}
	defer rows.Close()

	return pr.scanPlans(rows)
}

// FindActive returns all active plans
func (pr *PlanRepository) FindActive(ctx context.Context) ([]Plan, error) {
	query := sq.Select(planColumns...).
		From("plan").
		Where(sq.Eq{"is_active": true}).
		OrderBy("is_default DESC", "name ASC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active plans: %w", err)
	}
	defer rows.Close()

	return pr.scanPlans(rows)
}

// FindById returns a plan by ID
func (pr *PlanRepository) FindById(ctx context.Context, id int64) (*Plan, error) {
	query := sq.Select(planColumns...).
		From("plan").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	row := pr.pool.QueryRow(ctx, sql, args...)
	return pr.scanPlan(row)
}

// FindDefault returns the default plan
func (pr *PlanRepository) FindDefault(ctx context.Context) (*Plan, error) {
	query := sq.Select(planColumns...).
		From("plan").
		Where(sq.Eq{"is_default": true}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	row := pr.pool.QueryRow(ctx, sql, args...)
	return pr.scanPlan(row)
}

// Create creates a new plan
func (pr *PlanRepository) Create(ctx context.Context, plan *Plan) (*Plan, error) {
	query := `
		INSERT INTO plan (name, price_1, price_3, price_6, price_12, traffic_limit, device_limit, is_active, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, price_1, price_3, price_6, price_12, traffic_limit, device_limit, is_active, is_default, created_at, updated_at
	`

	row := pr.pool.QueryRow(ctx, query,
		plan.Name, plan.Price1, plan.Price3, plan.Price6, plan.Price12,
		plan.TrafficLimit, plan.DeviceLimit, plan.IsActive, plan.IsDefault,
	)

	return pr.scanPlan(row)
}

// Update updates an existing plan
func (pr *PlanRepository) Update(ctx context.Context, plan *Plan) (*Plan, error) {
	query := `
		UPDATE plan SET
			name = $2, price_1 = $3, price_3 = $4, price_6 = $5, price_12 = $6,
			traffic_limit = $7, device_limit = $8, is_active = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, price_1, price_3, price_6, price_12, traffic_limit, device_limit, is_active, is_default, created_at, updated_at
	`

	row := pr.pool.QueryRow(ctx, query,
		plan.ID, plan.Name, plan.Price1, plan.Price3, plan.Price6, plan.Price12,
		plan.TrafficLimit, plan.DeviceLimit, plan.IsActive,
	)

	return pr.scanPlan(row)
}

// Delete deletes a plan by ID (cannot delete default plan)
func (pr *PlanRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM plan WHERE id = $1 AND is_default = FALSE`
	
	result, err := pr.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("plan not found or is default plan")
	}

	return nil
}

// SetDefault sets a plan as default (unsets others)
func (pr *PlanRepository) SetDefault(ctx context.Context, id int64) error {
	tx, err := pr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Unset current default
	_, err = tx.Exec(ctx, "UPDATE plan SET is_default = FALSE WHERE is_default = TRUE")
	if err != nil {
		return fmt.Errorf("failed to unset default: %w", err)
	}

	// Set new default
	result, err := tx.Exec(ctx, "UPDATE plan SET is_default = TRUE WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("plan not found")
	}

	return tx.Commit(ctx)
}
