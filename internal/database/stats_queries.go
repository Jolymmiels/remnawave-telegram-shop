package database

import (
	"context"
	"remnawave-tg-shop-bot/internal/stats"

	"github.com/jackc/pgx/v4/pgxpool"
)

type StatsQueries struct {
	pool *pgxpool.Pool
}

func NewStatsQueries(pool *pgxpool.Pool) *StatsQueries {
	return &StatsQueries{pool: pool}
}

func (sq *StatsQueries) GetReferralStats(ctx context.Context) (*stats.ReferralStats, error) {
	s := &stats.ReferralStats{}

	// Total referrals
	err := sq.pool.QueryRow(ctx, `SELECT COUNT(*) FROM referral`).Scan(&s.TotalReferrals)
	if err != nil {
		return nil, err
	}

	// Active referrers (users who have at least 1 referral)
	err = sq.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT referrer_id) FROM referral`).Scan(&s.ActiveReferrers)
	if err != nil {
		return nil, err
	}

	// Total bonus days granted
	err = sq.pool.QueryRow(ctx, `SELECT COALESCE(SUM(bonus_days), 0) FROM referral_bonus_history`).Scan(&s.BonusDaysGranted)
	if err != nil {
		return nil, err
	}

	// Conversion rate: referees who made a purchase / total referees
	var totalReferees, paidReferees int64
	err = sq.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT referee_id) FROM referral`).Scan(&totalReferees)
	if err != nil {
		return nil, err
	}

	err = sq.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT r.referee_id) 
		FROM referral r
		JOIN customer c ON c.telegram_id = r.referee_id
		JOIN purchase p ON p.customer_id = c.id AND p.status = 'paid'
	`).Scan(&paidReferees)
	if err != nil {
		return nil, err
	}

	if totalReferees > 0 {
		s.ConversionRate = float64(paidReferees) / float64(totalReferees) * 100
	}

	return s, nil
}

func (sq *StatsQueries) GetPromoStats(ctx context.Context) (*stats.PromoStats, error) {
	s := &stats.PromoStats{}

	// Total promos
	err := sq.pool.QueryRow(ctx, `SELECT COUNT(*) FROM promo`).Scan(&s.TotalPromos)
	if err != nil {
		return nil, err
	}

	// Active promos
	err = sq.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM promo 
		WHERE active = true 
		AND (expires_at IS NULL OR expires_at > NOW())
		AND (max_uses IS NULL OR used_count < max_uses)
	`).Scan(&s.ActivePromos)
	if err != nil {
		return nil, err
	}

	// Total usages
	err = sq.pool.QueryRow(ctx, `SELECT COUNT(*) FROM promo_usage`).Scan(&s.TotalUsages)
	if err != nil {
		return nil, err
	}

	// Bonus days granted through promos
	err = sq.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.bonus_days), 0) 
		FROM promo_usage pu
		JOIN promo p ON p.id = pu.promo_id
	`).Scan(&s.BonusDaysGranted)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (sq *StatsQueries) GetPlanStats(ctx context.Context) ([]stats.PlanStats, error) {
	rows, err := sq.pool.Query(ctx, `
		SELECT 
			p.id,
			p.name,
			COUNT(pu.id) as count,
			COALESCE(SUM(pu.amount), 0) as amount
		FROM plan p
		LEFT JOIN purchase pu ON pu.plan_id = p.id AND pu.status = 'paid'
		GROUP BY p.id, p.name
		ORDER BY amount DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []stats.PlanStats
	var totalAmount float64

	for rows.Next() {
		var s stats.PlanStats
		if err := rows.Scan(&s.PlanID, &s.PlanName, &s.Count, &s.Amount); err != nil {
			return nil, err
		}
		totalAmount += s.Amount
		result = append(result, s)
	}

	// Calculate percentages
	for i := range result {
		if totalAmount > 0 {
			result[i].Percent = result[i].Amount / totalAmount * 100
		}
	}

	return result, nil
}

func (sq *StatsQueries) GetPeriodStats(ctx context.Context) ([]stats.PeriodStats, error) {
	rows, err := sq.pool.Query(ctx, `
		SELECT 
			month,
			COUNT(*) as count,
			COALESCE(SUM(amount), 0) as amount
		FROM purchase 
		WHERE status = 'paid'
		GROUP BY month
		ORDER BY month
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []stats.PeriodStats
	var totalAmount float64

	for rows.Next() {
		var s stats.PeriodStats
		if err := rows.Scan(&s.Months, &s.Count, &s.Amount); err != nil {
			return nil, err
		}
		totalAmount += s.Amount
		result = append(result, s)
	}

	// Calculate percentages
	for i := range result {
		if totalAmount > 0 {
			result[i].Percent = result[i].Amount / totalAmount * 100
		}
	}

	return result, nil
}

func (sq *StatsQueries) GetAutopayStats(ctx context.Context) (*stats.AutopayStats, error) {
	s := &stats.AutopayStats{}

	// Users with autopay enabled
	err := sq.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM customer WHERE autopay_enabled = true
	`).Scan(&s.EnabledUsers)
	if err != nil {
		return nil, err
	}

	// Users with payment method
	err = sq.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM customer WHERE payment_method_id IS NOT NULL
	`).Scan(&s.TotalWithMethod)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (sq *StatsQueries) GetTrialStats(ctx context.Context) (*stats.TrialStats, error) {
	s := &stats.TrialStats{}

	// Total users who used trial
	err := sq.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customer WHERE trial_used = true`).Scan(&s.TotalUsed)
	if err != nil {
		return nil, err
	}

	// Users who used trial and then made a purchase
	err = sq.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT c.id) 
		FROM customer c
		JOIN purchase p ON p.customer_id = c.id AND p.status = 'paid'
		WHERE c.trial_used = true
	`).Scan(&s.ConvertedToPaid)
	if err != nil {
		return nil, err
	}

	if s.TotalUsed > 0 {
		s.ConversionRate = float64(s.ConvertedToPaid) / float64(s.TotalUsed) * 100
	}

	return s, nil
}

func (sq *StatsQueries) GetLanguageStats(ctx context.Context) ([]stats.LanguageStat, error) {
	rows, err := sq.pool.Query(ctx, `
		SELECT 
			COALESCE(NULLIF(language, ''), 'en') as lang,
			COUNT(*) as count
		FROM customer
		GROUP BY lang
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []stats.LanguageStat
	var total int64

	for rows.Next() {
		var s stats.LanguageStat
		if err := rows.Scan(&s.Language, &s.Count); err != nil {
			return nil, err
		}
		total += s.Count
		result = append(result, s)
	}

	// Calculate percentages
	for i := range result {
		if total > 0 {
			result[i].Percent = float64(result[i].Count) / float64(total) * 100
		}
	}

	return result, nil
}
