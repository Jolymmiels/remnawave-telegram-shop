package database

import (
	"context"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

type Referral struct {
	ID           int64     `db:"id"`
	ReferrerID   int64     `db:"referrer_id"`
	RefereeID    int64     `db:"referee_id"`
	UsedAt       time.Time `db:"used_at"`
	BonusGranted bool      `db:"bonus_granted"`
}

type ReferralBonusHistory struct {
	ID           int64     `db:"id"`
	ReferralID   int64     `db:"referral_id"`
	PurchaseID   *int64    `db:"purchase_id"`
	BonusDays    int       `db:"bonus_days"`
	IsFirstBonus bool      `db:"is_first_bonus"`
	GrantedAt    time.Time `db:"granted_at"`
}

type ReferralRepository struct {
	pool *pgxpool.Pool
}

func NewReferralRepository(pool *pgxpool.Pool) *ReferralRepository {
	return &ReferralRepository{pool: pool}
}

func (r *ReferralRepository) Create(ctx context.Context, referrerID, refereeID int64) (*Referral, error) {
	query := sq.Insert("referral").
		Columns("referrer_id", "referee_id", "used_at", "bonus_granted").
		Values(referrerID, refereeID, sq.Expr("NOW()"), false).
		Suffix("RETURNING id, referrer_id, referee_id, used_at, bonus_granted").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert referral query: %w", err)
	}

	row := r.pool.QueryRow(ctx, sql, args...)
	var ref Referral
	if err := row.Scan(&ref.ID, &ref.ReferrerID, &ref.RefereeID, &ref.UsedAt, &ref.BonusGranted); err != nil {
		return nil, fmt.Errorf("failed to scan inserted referral: %w", err)
	}
	return &ref, nil
}

func (r *ReferralRepository) FindByReferrer(ctx context.Context, referrerID int64) ([]Referral, error) {
	query := sq.Select("id", "referrer_id", "referee_id", "used_at", "bonus_granted").
		From("referral").
		Where(sq.Eq{"referrer_id": referrerID}).
		OrderBy("used_at DESC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select referrals by referrer query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query referrals by referrer: %w", err)
	}
	defer rows.Close()

	var list []Referral
	for rows.Next() {
		var ref Referral
		if err := rows.Scan(&ref.ID, &ref.ReferrerID, &ref.RefereeID, &ref.UsedAt, &ref.BonusGranted); err != nil {
			return nil, fmt.Errorf("failed to scan referral row: %w", err)
		}
		list = append(list, ref)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating referral rows: %w", rows.Err())
	}
	return list, nil
}

func (r *ReferralRepository) CountByReferrer(ctx context.Context, referrerID int64) (int, error) {
	query := sq.Select("COUNT(*)").
		From("referral").
		Where(sq.Eq{"referrer_id": referrerID}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build count referrals by referrer query: %w", err)
	}

	var count int
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to scan count of referrals: %w", err)
	}
	return count, nil
}

func (r *ReferralRepository) FindByReferee(ctx context.Context, refereeID int64) (*Referral, error) {
	query := sq.Select("id", "referrer_id", "referee_id", "used_at", "bonus_granted").
		From("referral").
		Where(sq.Eq{"referee_id": refereeID}).
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select referral by referee query: %w", err)
	}

	var ref Referral
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&ref.ID, &ref.ReferrerID, &ref.RefereeID, &ref.UsedAt, &ref.BonusGranted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query referral by referee: %w", err)
	}
	return &ref, nil
}

func (r *ReferralRepository) MarkBonusGranted(ctx context.Context, referralID int64) error {
	query := sq.Update("referral").
		Set("bonus_granted", true).
		Where(sq.Eq{"id": referralID}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update bonus_granted query: %w", err)
	}

	res, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute update bonus_granted: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("no referral record updated")
	}
	return nil
}

func (r *ReferralRepository) CreateBonusHistory(ctx context.Context, referralID int64, purchaseID *int64, bonusDays int, isFirstBonus bool) (*ReferralBonusHistory, error) {
	query := sq.Insert("referral_bonus_history").
		Columns("referral_id", "purchase_id", "bonus_days", "is_first_bonus", "granted_at").
		Values(referralID, purchaseID, bonusDays, isFirstBonus, sq.Expr("NOW()")).
		Suffix("RETURNING id, referral_id, purchase_id, bonus_days, is_first_bonus, granted_at").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert referral_bonus_history query: %w", err)
	}

	row := r.pool.QueryRow(ctx, sql, args...)
	var h ReferralBonusHistory
	if err := row.Scan(&h.ID, &h.ReferralID, &h.PurchaseID, &h.BonusDays, &h.IsFirstBonus, &h.GrantedAt); err != nil {
		return nil, fmt.Errorf("failed to scan inserted referral_bonus_history: %w", err)
	}
	return &h, nil
}

func (r *ReferralRepository) GetBonusHistoryByReferrer(ctx context.Context, referrerID int64) ([]ReferralBonusHistory, error) {
	query := sq.Select("rbh.id", "rbh.referral_id", "rbh.purchase_id", "rbh.bonus_days", "rbh.is_first_bonus", "rbh.granted_at").
		From("referral_bonus_history rbh").
		Join("referral ref ON ref.id = rbh.referral_id").
		Where(sq.Eq{"ref.referrer_id": referrerID}).
		OrderBy("rbh.granted_at DESC").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select referral_bonus_history query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query referral_bonus_history: %w", err)
	}
	defer rows.Close()

	var list []ReferralBonusHistory
	for rows.Next() {
		var h ReferralBonusHistory
		if err := rows.Scan(&h.ID, &h.ReferralID, &h.PurchaseID, &h.BonusDays, &h.IsFirstBonus, &h.GrantedAt); err != nil {
			return nil, fmt.Errorf("failed to scan referral_bonus_history row: %w", err)
		}
		list = append(list, h)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating referral_bonus_history rows: %w", rows.Err())
	}
	return list, nil
}

func (r *ReferralRepository) SumBonusDaysByReferrer(ctx context.Context, referrerID int64) (int, error) {
	query := sq.Select("COALESCE(SUM(rbh.bonus_days), 0)").
		From("referral_bonus_history rbh").
		Join("referral ref ON ref.id = rbh.referral_id").
		Where(sq.Eq{"ref.referrer_id": referrerID}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build sum bonus_days query: %w", err)
	}

	var sum int
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&sum); err != nil {
		return 0, fmt.Errorf("failed to scan sum of bonus_days: %w", err)
	}
	return sum, nil
}
