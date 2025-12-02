package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/stats"
	"strconv"
	"time"
)

type StatsHandler struct {
	purchaseRepository *database.PurchaseRepository
	customerRepository *database.CustomerRepository
	statsQueries       *database.StatsQueries
}

func NewStatsHandler(purchaseRepository *database.PurchaseRepository, customerRepository *database.CustomerRepository, statsQueries *database.StatsQueries) *StatsHandler {
	return &StatsHandler{
		purchaseRepository: purchaseRepository,
		customerRepository: customerRepository,
		statsQueries:       statsQueries,
	}
}

func (sh *StatsHandler) GetStatsTotals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Неделя начинается с понедельника
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	startOfWeek := startOfDay.AddDate(0, 0, offset)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	totals := stats.StatsTotals{}

	var err error
	totals.Day, err = sh.purchaseRepository.GetTotalAmountByDateRange(ctx, startOfDay, now)
	if err != nil {
		slog.Error("Failed to get daily total", "error", err)
		http.Error(w, "Failed to get daily total", http.StatusInternalServerError)
		return
	}

	totals.Week, err = sh.purchaseRepository.GetTotalAmountByDateRange(ctx, startOfWeek, now)
	if err != nil {
		slog.Error("Failed to get weekly total", "error", err)
		http.Error(w, "Failed to get weekly total", http.StatusInternalServerError)
		return
	}

	totals.Month, err = sh.purchaseRepository.GetTotalAmountByDateRange(ctx, startOfMonth, now)
	if err != nil {
		slog.Error("Failed to get monthly total", "error", err)
		http.Error(w, "Failed to get monthly total", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(totals)
}

func (sh *StatsHandler) GetMonthlyGrowth(w http.ResponseWriter, r *http.Request) {
	// if !isAdminRequest(r) {
	//     http.Error(w, "Forbidden", http.StatusForbidden)
	//     return
	// }

	ctx := r.Context()

	growthData, err := sh.purchaseRepository.GetMonthlyGrowthLastYear(ctx)
	if err != nil {
		slog.Error("Failed to get monthly growth", "error", err)
		http.Error(w, "Failed to get monthly growth", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(growthData)
}

func (sh *StatsHandler) GetUserByTelegramID(w http.ResponseWriter, r *http.Request) {
	// TODO: Добавить проверку авторизации администратора

	// Извлечение Telegram ID из URL
	// Для Go 1.22+:
	telegramIDStr := r.PathValue("telegramID") // Используем PathValue для Go 1.22+
	// Для gorilla/mux: vars := mux.Vars(r); telegramIDStr := vars["telegramID"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		slog.Warn("Invalid Telegram ID provided", "input", telegramIDStr)
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Используем метод репозитория для поиска пользователя
	customer, err := sh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to find customer by Telegram ID", "error", err, "telegramID", telegramID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if customer == nil {
		slog.Info("Customer not found by Telegram ID", "telegramID", telegramID)
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// data, err := json.MarshalIndent(customer, "", "  ")
	data, err := json.Marshal(customer)
	if err != nil {
		slog.Error("Failed to marshal customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// GetUserGrowthStats обрабатывает запрос GET /api/users/stats/growth
// Возвращает статистику роста пользователей.
func (sh *StatsHandler) GetUserGrowthStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := sh.customerRepository.GetUserGrowthStats(ctx)
	if err != nil {
		slog.Error("Failed to get user growth stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(stats)
	if err != nil {
		slog.Error("Failed to marshal user growth stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func (sh *StatsHandler) GetStatsOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userStats, err := sh.customerRepository.GetUserStats(ctx)
	if err != nil {
		slog.Error("Failed to get user stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	revenueStats, err := sh.purchaseRepository.GetRevenueStats(ctx)
	if err != nil {
		slog.Error("Failed to get revenue stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	paymentStats, err := sh.purchaseRepository.GetPaymentStats(ctx)
	if err != nil {
		slog.Error("Failed to get payment stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	referralStats, err := sh.statsQueries.GetReferralStats(ctx)
	if err != nil {
		slog.Error("Failed to get referral stats", "error", err)
		referralStats = &stats.ReferralStats{}
	}

	promoStats, err := sh.statsQueries.GetPromoStats(ctx)
	if err != nil {
		slog.Error("Failed to get promo stats", "error", err)
		promoStats = &stats.PromoStats{}
	}

	planStats, err := sh.statsQueries.GetPlanStats(ctx)
	if err != nil {
		slog.Error("Failed to get plan stats", "error", err)
		planStats = []stats.PlanStats{}
	}

	periodStats, err := sh.statsQueries.GetPeriodStats(ctx)
	if err != nil {
		slog.Error("Failed to get period stats", "error", err)
		periodStats = []stats.PeriodStats{}
	}

	autopayStats, err := sh.statsQueries.GetAutopayStats(ctx)
	if err != nil {
		slog.Error("Failed to get autopay stats", "error", err)
		autopayStats = &stats.AutopayStats{}
	}

	trialStats, err := sh.statsQueries.GetTrialStats(ctx)
	if err != nil {
		slog.Error("Failed to get trial stats", "error", err)
		trialStats = &stats.TrialStats{}
	}

	languageStats, err := sh.statsQueries.GetLanguageStats(ctx)
	if err != nil {
		slog.Error("Failed to get language stats", "error", err)
		languageStats = []stats.LanguageStat{}
	}

	overview := stats.StatsOverview{
		Users:     *userStats,
		Revenue:   *revenueStats,
		Payments:  *paymentStats,
		Referrals: *referralStats,
		Promos:    *promoStats,
		Plans:     planStats,
		Periods:   periodStats,
		Autopay:   *autopayStats,
		Trial:     *trialStats,
		Languages: languageStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func (sh *StatsHandler) GetDailyUserGrowth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	growth, err := sh.customerRepository.GetDailyUserGrowth(ctx, days)
	if err != nil {
		slog.Error("Failed to get daily user growth", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(growth)
}

func (sh *StatsHandler) GetDailyRevenue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	revenue, err := sh.purchaseRepository.GetDailyRevenue(ctx, days)
	if err != nil {
		slog.Error("Failed to get daily revenue", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(revenue)
}
