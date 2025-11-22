package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"remnawave-tg-shop-bot/internal/config"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AuthHandler struct {
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

type CheckAdminRequest struct {
	TelegramID int64 `json:"telegram_id"`
}

type CheckAdminResponse struct {
	IsAdmin    bool   `json:"is_admin"`
	TelegramID int64  `json:"telegram_id"`
	Message    string `json:"message,omitempty"`
}

// CheckAdmin verifies if the requesting user is an admin
func (ah *AuthHandler) CheckAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate Telegram init data
	initData := r.Header.Get("Telegram-Init-Data")
	if initData == "" {
		http.Error(w, "Telegram init data required", http.StatusUnauthorized)
		return
	}

	// Validate the Telegram WebApp init data signature
	telegramID, err := ah.validateTelegramInitData(initData)
	if err != nil {
		slog.Error("Invalid Telegram init data", "error", err)
		http.Error(w, "Invalid Telegram authentication", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req CheckAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify that the requesting user matches the init data
	if telegramID != req.TelegramID {
		slog.Warn("Telegram ID mismatch", "initData", telegramID, "request", req.TelegramID)
		http.Error(w, "Authentication mismatch", http.StatusUnauthorized)
		return
	}

	// Check if user is admin
	isAdmin := ah.isUserAdmin(ctx, telegramID)

	response := CheckAdminResponse{
		IsAdmin:    isAdmin,
		TelegramID: telegramID,
	}

	if isAdmin {
		response.Message = "Administrator access granted"
		slog.Info("Admin access granted", "telegramID", telegramID)
	} else {
		response.Message = "Administrator privileges required"
		slog.Warn("Admin access denied", "telegramID", telegramID)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode admin check response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// validateTelegramInitData validates the Telegram WebApp init data signature
func (ah *AuthHandler) validateTelegramInitData(initData string) (int64, error) {
	// Parse the init data
	values, err := url.ParseQuery(initData)
	if err != nil {
		return 0, fmt.Errorf("failed to parse init data: %w", err)
	}

	// Extract hash and remove it from values for verification
	hash := values.Get("hash")
	if hash == "" {
		return 0, fmt.Errorf("missing hash in init data")
	}
	values.Del("hash")

	// Create data check string
	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckPairs []string
	for _, k := range keys {
		dataCheckPairs = append(dataCheckPairs, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(dataCheckPairs, "\n")

	// Create secret key
	botToken := config.TelegramToken()
	if botToken == "" {
		return 0, fmt.Errorf("bot token not configured")
	}

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Calculate expected hash
	expectedHash := hmac.New(sha256.New, secretKey.Sum(nil))
	expectedHash.Write([]byte(dataCheckString))
	expectedHashString := hex.EncodeToString(expectedHash.Sum(nil))

	// Verify hash
	if hash != expectedHashString {
		return 0, fmt.Errorf("invalid signature")
	}

	// Check auth_date (data should not be older than 1 hour)
	authDateStr := values.Get("auth_date")
	if authDateStr == "" {
		return 0, fmt.Errorf("missing auth_date")
	}

	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid auth_date: %w", err)
	}

	// Check if data is not too old (1 hour = 3600 seconds)
	now := time.Now().Unix()
	if now-authDate > 3600 {
		return 0, fmt.Errorf("init data expired")
	}

	// Extract user ID
	userStr := values.Get("user")
	if userStr == "" {
		return 0, fmt.Errorf("missing user data")
	}

	var userData struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userStr), &userData); err != nil {
		return 0, fmt.Errorf("failed to parse user data: %w", err)
	}

	if userData.ID == 0 {
		return 0, fmt.Errorf("invalid user ID")
	}

	return userData.ID, nil
}

// isUserAdmin checks if the given Telegram ID is an admin
func (ah *AuthHandler) isUserAdmin(ctx context.Context, telegramID int64) bool {
	// Get admin Telegram ID from environment
	adminTelegramID := config.GetAdminTelegramId()
	if adminTelegramID == 0 {
		slog.Error("Admin Telegram ID not configured")
		return false
	}

	// Simple check - in production you might want to check against a database
	return telegramID == adminTelegramID
}

// Middleware to verify admin access for protected routes
func (ah *AuthHandler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate Telegram init data
		initData := r.Header.Get("Telegram-Init-Data")
		if initData == "" {
			slog.Warn("Missing Telegram init data", "path", r.URL.Path, "method", r.Method)
			http.Error(w, "Telegram authentication required", http.StatusUnauthorized)
			return
		}

		// Validate the Telegram WebApp init data signature
		telegramID, err := ah.validateTelegramInitData(initData)
		if err != nil {
			slog.Error("Invalid Telegram init data", "error", err, "path", r.URL.Path)
			http.Error(w, "Invalid Telegram authentication", http.StatusUnauthorized)
			return
		}

		// Check if user is admin
		if !ah.isUserAdmin(r.Context(), telegramID) {
			slog.Warn("Non-admin access attempt", "telegramID", telegramID, "path", r.URL.Path)
			http.Error(w, "Administrator privileges required", http.StatusForbidden)
			return
		}

		// Add telegram ID to request context for use in handlers
		ctx := context.WithValue(r.Context(), "telegram_id", telegramID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
