package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"private-remnawave-telegram-shop-bot/internal/database"
	"strconv"
	"strings"
)

type UsersHandler struct {
	customerRepository *database.CustomerRepository
	purchaseRepository *database.PurchaseRepository
	referralRepository *database.ReferralRepository
}

func NewUsersHandler(customerRepository *database.CustomerRepository, purchaseRepository *database.PurchaseRepository, referralRepository *database.ReferralRepository) *UsersHandler {
	return &UsersHandler{
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		referralRepository: referralRepository,
	}
}

type UserSearchResponse struct {
	Users []UserWithDetails `json:"users"`
	Total int               `json:"total"`
}

type UserWithDetails struct {
	database.Customer
	PaymentsCount  int     `json:"payments_count"`
	ReferralsCount int     `json:"referrals_count"`
	TotalSpent     float64 `json:"total_spent"`
}

type UserPaymentHistory struct {
	database.Purchase
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
		userDetails := UserWithDetails{
			Customer: customer,
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payments); err != nil {
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

	// For now, we'll "soft delete" by setting expire_at to past date
	// In a production system, you might want proper soft deletion or cascading deletes
	updates := map[string]interface{}{
		"expire_at": "1970-01-01 00:00:00",
	}

	if err := uh.customerRepository.UpdateFields(ctx, customer.ID, updates); err != nil {
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

	// Block user by expiring their subscription
	updates := map[string]interface{}{
		"expire_at": "1970-01-01 00:00:00",
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

	// Parse request body for new expiration date
	var requestData struct {
		ExpireAt *string `json:"expire_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"expire_at": requestData.ExpireAt,
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
