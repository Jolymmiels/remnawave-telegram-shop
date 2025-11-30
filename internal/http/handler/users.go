package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"strconv"
	"strings"
)

type UsersHandler struct {
	customerRepository *database.CustomerRepository
	purchaseRepository *database.PurchaseRepository
	referralRepository *database.ReferralRepository
	planRepository     *database.PlanRepository
	remnawaveClient    *remnawave.Client
}

func NewUsersHandler(customerRepository *database.CustomerRepository, purchaseRepository *database.PurchaseRepository, referralRepository *database.ReferralRepository, planRepository *database.PlanRepository, remnawaveClient *remnawave.Client) *UsersHandler {
	return &UsersHandler{
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		referralRepository: referralRepository,
		planRepository:     planRepository,
		remnawaveClient:    remnawaveClient,
	}
}

type UserSearchResponse struct {
	Users []UserWithDetails `json:"users"`
	Total int               `json:"total"`
}

type UserWithDetails struct {
	ID               int64   `json:"id"`
	TelegramID       int64   `json:"telegram_id"`
	ExpireAt         *string `json:"expire_at"`
	CreatedAt        string  `json:"created_at"`
	SubscriptionLink *string `json:"subscription_link"`
	Language         string  `json:"language"`
	IsBlocked        bool    `json:"is_blocked"`
	PaymentsCount    int     `json:"payments_count"`
	ReferralsCount   int     `json:"referrals_count"`
	TotalSpent       float64 `json:"total_spent"`
}

type PaymentDTO struct {
	ID                int64   `json:"id"`
	Amount            float64 `json:"amount"`
	CustomerID        int64   `json:"customer_id"`
	CreatedAt         string  `json:"created_at"`
	Month             int     `json:"month"`
	PaidAt            *string `json:"paid_at"`
	Currency          string  `json:"currency"`
	ExpireAt          *string `json:"expire_at"`
	Status            string  `json:"status"`
	InvoiceType       string  `json:"invoice_type"`
	CryptoInvoiceID   *int64  `json:"crypto_invoice_id"`
	CryptoInvoiceLink *string `json:"crypto_invoice_link"`
	YookasaURL        *string `json:"yookasa_url"`
	YookasaID         *string `json:"yookasa_id"`
	PlanID            *int64  `json:"plan_id"`
	PlanName          *string `json:"plan_name"`
}

// SearchUsers handles GET /api/users/search?q=query&limit=20&offset=0&sort=date&order=desc&status=active
func (uh *UsersHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	sortBy := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	status := r.URL.Query().Get("status")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse sort parameters
	var dbSortBy database.CustomerSortField
	switch sortBy {
	case "spent":
		dbSortBy = database.SortBySpent
	case "referrals":
		dbSortBy = database.SortByReferrals
	default:
		dbSortBy = database.SortByDate
	}

	var dbSortOrder database.CustomerSortOrder
	if sortOrder == "asc" {
		dbSortOrder = database.SortAsc
	} else {
		dbSortOrder = database.SortDesc
	}

	// Parse status filter
	var dbStatus database.CustomerStatusFilter
	switch status {
	case "active":
		dbStatus = database.StatusActive
	case "expired":
		dbStatus = database.StatusExpired
	case "no_subscription":
		dbStatus = database.StatusNoSubscription
	default:
		dbStatus = database.StatusAll
	}

	// Use unified search with all filters
	params := database.CustomerSearchParams{
		Query:     query,
		Status:    dbStatus,
		SortBy:    dbSortBy,
		SortOrder: dbSortOrder,
		Limit:     limit,
		Offset:    offset,
	}

	customersWithStats, totalCount, err := uh.customerRepository.FindAllSorted(ctx, params)
	if err != nil {
		slog.Error("Failed to get customers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	usersWithDetails := make([]UserWithDetails, 0, len(customersWithStats))
	for _, c := range customersWithStats {
		var expireAtStr *string
		if c.ExpireAt != nil {
			expireAtStr = stringPtr(c.ExpireAt.Format("2006-01-02T15:04:05Z07:00"))
		}

		usersWithDetails = append(usersWithDetails, UserWithDetails{
			ID:               c.ID,
			TelegramID:       c.TelegramID,
			ExpireAt:         expireAtStr,
			CreatedAt:        c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			SubscriptionLink: c.SubscriptionLink,
			Language:         c.Language,
			IsBlocked:        c.IsBlocked,
			PaymentsCount:    c.PaymentsCount,
			ReferralsCount:   c.ReferralsCount,
			TotalSpent:       c.TotalSpent,
		})
	}

	response := UserSearchResponse{
		Users: usersWithDetails,
		Total: totalCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetUserPayments handles GET /api/users/{telegramID}/payments
func (uh *UsersHandler) GetUserPayments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Find customer first
	customer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to find customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get payments
	payments, err := uh.getCustomerPayments(ctx, customer.ID)
	if err != nil {
		slog.Error("Failed to get customer payments", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build plan name cache
	planNames := make(map[int64]string)
	for _, p := range payments {
		if p.PlanID != nil {
			if _, ok := planNames[*p.PlanID]; !ok {
				if plan, err := uh.planRepository.FindById(ctx, *p.PlanID); err == nil && plan != nil {
					planNames[*p.PlanID] = plan.Name
				}
			}
		}
	}

	// Convert to DTOs with snake_case fields
	paymentDTOs := make([]PaymentDTO, len(payments))
	for i, p := range payments {
		paidAtStr := ""
		if p.PaidAt != nil {
			paidAtStr = p.PaidAt.Format("2006-01-02T15:04:05Z07:00")
		}
		expireAtStr := ""
		if p.ExpireAt != nil {
			expireAtStr = p.ExpireAt.Format("2006-01-02T15:04:05Z07:00")
		}

		yookasaIDStr := ""
		if p.YookasaID != nil {
			yookasaIDStr = p.YookasaID.String()
		}

		var planName *string
		if p.PlanID != nil {
			if name, ok := planNames[*p.PlanID]; ok {
				planName = &name
			}
		}

		paymentDTOs[i] = PaymentDTO{
			ID:                p.ID,
			Amount:            p.Amount,
			CustomerID:        p.CustomerID,
			CreatedAt:         p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Month:             p.Month,
			PaidAt:            stringPtrIfNotEmpty(paidAtStr),
			Currency:          p.Currency,
			ExpireAt:          stringPtrIfNotEmpty(expireAtStr),
			Status:            string(p.Status),
			InvoiceType:       string(p.InvoiceType),
			CryptoInvoiceID:   p.CryptoInvoiceID,
			CryptoInvoiceLink: p.CryptoInvoiceLink,
			YookasaURL:        p.YookasaURL,
			YookasaID:         stringPtrIfNotEmpty(yookasaIDStr),
			PlanID:            p.PlanID,
			PlanName:          planName,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(paymentDTOs); err != nil {
		slog.Error("Failed to encode payments", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// UpdateUser handles PUT /api/users/{telegramID}
func (uh *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Find customer first
	customer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to find customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate and filter allowed fields
	allowedFields := map[string]bool{
		"language":          true,
		"expire_at":         true,
		"subscription_link": true,
	}

	updates := make(map[string]interface{})
	for field, value := range updateData {
		if allowedFields[field] {
			updates[field] = value
		}
	}

	if len(updates) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	// Update user
	if err := uh.customerRepository.UpdateFields(ctx, customer.ID, updates); err != nil {
		slog.Error("Failed to update customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return updated customer
	updatedCustomer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get updated customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedCustomer); err != nil {
		slog.Error("Failed to encode updated customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// DeleteUser handles DELETE /api/users/{telegramID}
func (uh *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Delete customer from database
	if err := uh.customerRepository.DeleteByTelegramId(ctx, telegramID); err != nil {
		slog.Error("Failed to delete customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BlockUser handles POST /api/users/{telegramID}/block
func (uh *UsersHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Find customer first
	customer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to find customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Block user using is_blocked flag
	updates := map[string]interface{}{
		"is_blocked": true,
	}

	if err := uh.customerRepository.UpdateFields(ctx, customer.ID, updates); err != nil {
		slog.Error("Failed to block customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnblockUser handles POST /api/users/{telegramID}/unblock
func (uh *UsersHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Find customer first
	customer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to find customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Unblock user using is_blocked flag
	updates := map[string]interface{}{
		"is_blocked": false,
	}

	if err := uh.customerRepository.UpdateFields(ctx, customer.ID, updates); err != nil {
		slog.Error("Failed to unblock customer", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper function to get customer payments
func (uh *UsersHandler) getCustomerPayments(ctx context.Context, customerID int64) ([]database.Purchase, error) {
	return uh.purchaseRepository.FindByCustomerID(ctx, customerID)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Helper function to create string pointer if not empty
func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// DeviceDTO represents a device in the API response
// ReferralBonusDTO represents a referral bonus in the API response
type ReferralBonusDTO struct {
	ID           int64   `json:"id"`
	ReferralID   int64   `json:"referral_id"`
	PurchaseID   *int64  `json:"purchase_id"`
	BonusDays    int     `json:"bonus_days"`
	IsFirstBonus bool    `json:"is_first_bonus"`
	GrantedAt    string  `json:"granted_at"`
	RefereeTgID  int64   `json:"referee_telegram_id"`
}

type DeviceDTO struct {
	Hwid        string `json:"hwid"`
	UserUuid    string `json:"user_uuid"`
	Platform    string `json:"platform"`
	OsVersion   string `json:"os_version"`
	DeviceModel string `json:"device_model"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// GetUserDevices handles GET /api/users/{telegramID}/devices
func (uh *UsersHandler) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	if uh.remnawaveClient == nil {
		http.Error(w, "Remnawave client not configured", http.StatusServiceUnavailable)
		return
	}

	userUuid, err := uh.remnawaveClient.GetUserUuidByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get user UUID", "error", err, "telegramID", telegramID)
		http.Error(w, "User not found in Remnawave", http.StatusNotFound)
		return
	}

	devices, err := uh.remnawaveClient.GetUserDevices(ctx, userUuid)
	if err != nil {
		slog.Error("Failed to get user devices", "error", err)
		http.Error(w, "Failed to get devices", http.StatusInternalServerError)
		return
	}

	deviceDTOs := make([]DeviceDTO, len(devices))
	for i, d := range devices {
		deviceDTOs[i] = DeviceDTO{
			Hwid:        d.Hwid,
			UserUuid:    d.UserUuid,
			Platform:    d.Platform,
			OsVersion:   d.OsVersion,
			DeviceModel: d.DeviceModel,
			CreatedAt:   d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deviceDTOs); err != nil {
		slog.Error("Failed to encode devices", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// DeleteUserDevice handles DELETE /api/users/{telegramID}/devices/{hwid}
func (uh *UsersHandler) DeleteUserDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	hwid := r.PathValue("hwid")
	if hwid == "" {
		http.Error(w, "HWID is required", http.StatusBadRequest)
		return
	}

	if uh.remnawaveClient == nil {
		http.Error(w, "Remnawave client not configured", http.StatusServiceUnavailable)
		return
	}

	userUuid, err := uh.remnawaveClient.GetUserUuidByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get user UUID", "error", err, "telegramID", telegramID)
		http.Error(w, "User not found in Remnawave", http.StatusNotFound)
		return
	}

	if err := uh.remnawaveClient.DeleteUserDevice(ctx, userUuid, hwid); err != nil {
		slog.Error("Failed to delete device", "error", err, "hwid", hwid)
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RevokeSubscription handles POST /api/users/{telegramID}/revoke-subscription
func (uh *UsersHandler) RevokeSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	if uh.remnawaveClient == nil {
		http.Error(w, "Remnawave client not configured", http.StatusServiceUnavailable)
		return
	}

	userUuid, err := uh.remnawaveClient.GetUserUuidByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get user UUID", "error", err, "telegramID", telegramID)
		http.Error(w, "User not found in Remnawave", http.StatusNotFound)
		return
	}

	newSubscriptionLink, err := uh.remnawaveClient.RevokeUserSubscription(ctx, userUuid)
	if err != nil {
		slog.Error("Failed to revoke subscription", "error", err, "telegramID", telegramID)
		http.Error(w, "Failed to revoke subscription", http.StatusInternalServerError)
		return
	}

	// Update local database - clear expire_at and set new subscription_link
	customer, err := uh.customerRepository.FindByTelegramId(ctx, telegramID)
	if err == nil && customer != nil {
		updates := map[string]interface{}{
			"subscription_link": newSubscriptionLink,
		}
		_ = uh.customerRepository.UpdateFields(ctx, customer.ID, updates)
	}

	slog.Info("Subscription revoked", "telegramID", telegramID, "newSubscriptionLink", newSubscriptionLink)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscription_link": newSubscriptionLink,
	})
}

// GetUserReferralBonuses handles GET /api/users/{telegramID}/referral-bonuses
func (uh *UsersHandler) GetUserReferralBonuses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Telegram ID", http.StatusBadRequest)
		return
	}

	// Get bonus history for this user as referrer
	bonuses, err := uh.referralRepository.GetBonusHistoryByReferrer(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get referral bonuses", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get referrals to map referral_id to referee telegram_id
	referrals, err := uh.referralRepository.FindByReferrer(ctx, telegramID)
	if err != nil {
		slog.Error("Failed to get referrals", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build referral ID to referee telegram ID map
	referralToReferee := make(map[int64]int64)
	for _, ref := range referrals {
		referralToReferee[ref.ID] = ref.RefereeID
	}

	// Convert to DTOs
	bonusDTOs := make([]ReferralBonusDTO, len(bonuses))
	for i, b := range bonuses {
		bonusDTOs[i] = ReferralBonusDTO{
			ID:           b.ID,
			ReferralID:   b.ReferralID,
			PurchaseID:   b.PurchaseID,
			BonusDays:    b.BonusDays,
			IsFirstBonus: b.IsFirstBonus,
			GrantedAt:    b.GrantedAt.Format("2006-01-02T15:04:05Z07:00"),
			RefereeTgID:  referralToReferee[b.ReferralID],
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bonusDTOs); err != nil {
		slog.Error("Failed to encode referral bonuses", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
