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
	remnawaveClient    *remnawave.Client
}

func NewUsersHandler(customerRepository *database.CustomerRepository, purchaseRepository *database.PurchaseRepository, referralRepository *database.ReferralRepository, remnawaveClient *remnawave.Client) *UsersHandler {
	return &UsersHandler{
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		referralRepository: referralRepository,
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
	ID              int64   `json:"id"`
	Amount          float64 `json:"amount"`
	CustomerID      int64   `json:"customer_id"`
	CreatedAt       string  `json:"created_at"`
	Month           int     `json:"month"`
	PaidAt          *string `json:"paid_at"`
	Currency        string  `json:"currency"`
	ExpireAt        *string `json:"expire_at"`
	Status          string  `json:"status"`
	InvoiceType     string  `json:"invoice_type"`
	CryptoInvoiceID *int64  `json:"crypto_invoice_id"`
	CryptoInvoiceLink *string `json:"crypto_invoice_link"`
	YookasaURL      *string `json:"yookasa_url"`
	YookasaID       *string `json:"yookasa_id"`
}

// SearchUsers handles GET /api/users/search?q=query&limit=20&offset=0
func (uh *UsersHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var customers []database.Customer

	if query != "" && strings.TrimSpace(query) != "" {
		// Try to parse as Telegram ID first
		if telegramID, parseErr := strconv.ParseInt(strings.TrimSpace(query), 10, 64); parseErr == nil {
			customer, findErr := uh.customerRepository.FindByTelegramId(ctx, telegramID)
			if findErr != nil {
				slog.Error("Failed to search user by telegram ID", "error", findErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if customer != nil {
				customers = []database.Customer{*customer}
			}
		} else {
			// If not a valid telegram ID, return empty results for now
			// In the future, we could search by username or other fields
			customers = []database.Customer{}
		}
	} else {
		// No query - return all users with pagination
		allCustomers, err := uh.customerRepository.FindAll(ctx)
		if err != nil {
			slog.Error("Failed to get all customers", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if allCustomers != nil {
			// Simple pagination
			start := offset
			end := offset + limit
			if start < len(*allCustomers) {
				if end > len(*allCustomers) {
					end = len(*allCustomers)
				}
				customers = (*allCustomers)[start:end]
			}
		}
	}

	// Enrich with additional details
	usersWithDetails := make([]UserWithDetails, 0, len(customers))
	for _, customer := range customers {
		var expireAtStr *string
		if customer.ExpireAt != nil {
			expireAtStr = stringPtr(customer.ExpireAt.Format("2006-01-02T15:04:05Z07:00"))
		}
		
		userDetails := UserWithDetails{
			ID:               customer.ID,
			TelegramID:       customer.TelegramID,
			ExpireAt:         expireAtStr,
			CreatedAt:        customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			SubscriptionLink: customer.SubscriptionLink,
			Language:         customer.Language,
			IsBlocked:        customer.IsBlocked,
		}

		// Get payments count and total spent
		if payments, err := uh.getCustomerPayments(ctx, customer.ID); err == nil {
			userDetails.PaymentsCount = len(payments)
			for _, payment := range payments {
				if payment.Status == database.PurchaseStatusPaid {
					userDetails.TotalSpent += payment.Amount
				}
			}
		}

		// Get referrals count
		if count, err := uh.referralRepository.CountByReferrer(ctx, customer.TelegramID); err == nil {
			userDetails.ReferralsCount = count
		}

		usersWithDetails = append(usersWithDetails, userDetails)
	}

	// For total count - this is simplified, in production you'd want a proper count query
	totalCount := len(usersWithDetails)
	if query == "" {
		// If no search query, get actual total from database
		if allCustomers, err := uh.customerRepository.FindAll(ctx); err == nil && allCustomers != nil {
			totalCount = len(*allCustomers)
		}
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
		
		paymentDTOs[i] = PaymentDTO{
			ID:              p.ID,
			Amount:          p.Amount,
			CustomerID:      p.CustomerID,
			CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Month:           p.Month,
			PaidAt:          stringPtrIfNotEmpty(paidAtStr),
			Currency:        p.Currency,
			ExpireAt:        stringPtrIfNotEmpty(expireAtStr),
			Status:          string(p.Status),
			InvoiceType:     string(p.InvoiceType),
			CryptoInvoiceID: p.CryptoInvoiceID,
			CryptoInvoiceLink: p.CryptoInvoiceLink,
			YookasaURL:      p.YookasaURL,
			YookasaID:       stringPtrIfNotEmpty(yookasaIDStr),
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
