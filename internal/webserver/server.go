package webserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/telegramlink"
	"remnawave-tg-shop-bot/internal/webauth"
	"remnawave-tg-shop-bot/internal/yookasa"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const sessionCookieName = "remnawave_session"

type Server struct {
	pool                *pgxpool.Pool
	remnawaveClient     *remnawave.Client
	customerRepository  *database.CustomerRepository
	purchaseRepository  *database.PurchaseRepository
	authService         *webauth.Service
	paymentService      *payment.PaymentService
	yookasaClient       *yookasa.Client
	telegramLinkService *telegramlink.Service
}

func New(
	pool *pgxpool.Pool,
	remnawaveClient *remnawave.Client,
	customerRepository *database.CustomerRepository,
	purchaseRepository *database.PurchaseRepository,
	authService *webauth.Service,
	paymentService *payment.PaymentService,
	yookasaClient *yookasa.Client,
	telegramLinkService *telegramlink.Service,
) *Server {
	return &Server{
		pool:                pool,
		remnawaveClient:     remnawaveClient,
		customerRepository:  customerRepository,
		purchaseRepository:  purchaseRepository,
		authService:         authService,
		paymentService:      paymentService,
		yookasaClient:       yookasaClient,
		telegramLinkService: telegramLinkService,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", s.healthcheck)
	mux.HandleFunc("/api/v1/plans", s.plans)
	mux.HandleFunc("/api/v1/auth/session", s.session)
	mux.HandleFunc("/api/v1/auth/register", s.register)
	mux.HandleFunc("/api/v1/auth/login", s.login)
	mux.HandleFunc("/api/v1/auth/logout", s.logout)
	mux.HandleFunc("/api/v1/auth/link-telegram", s.linkTelegram)
	mux.HandleFunc("/api/v1/checkout/yookassa", s.checkoutYookassa)
	mux.HandleFunc("/api/v1/purchases/status", s.purchaseStatus)
	mux.HandleFunc("/api/v1/trial/activate", s.activateTrial)

	return s.withCORS(mux)
}

func (s *Server) healthcheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{
		"status":    "ok",
		"db":        "ok",
		"remnawave": "ok",
		"time":      time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.pool.Ping(ctx); err != nil {
		status["status"] = "fail"
		status["db"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := s.remnawaveClient.Ping(ctx); err != nil {
		status["status"] = "fail"
		status["remnawave"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	writeJSON(w, status)
}

func (s *Server) plans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"plans": []map[string]any{
			{"months": 1, "price": config.Price(1), "currency": "RUB"},
			{"months": 3, "price": config.Price(3), "currency": "RUB"},
			{"months": 6, "price": config.Price(6), "currency": "RUB"},
			{"months": 12, "price": config.Price(12), "currency": "RUB"},
		},
		"trialDays": config.TrialDays(),
	})
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	customer, err := s.currentCustomer(r)
	if err != nil {
		writeJSON(w, map[string]any{
			"authenticated": false,
			"mode":          "web",
		})
		return
	}

	if err := s.syncCustomerPendingYookassaPurchases(r.Context(), customer); err == nil {
		if freshCustomer, freshErr := s.customerRepository.FindById(r.Context(), customer.ID); freshErr == nil && freshCustomer != nil {
			customer = freshCustomer
		}
	}

	writeJSON(w, map[string]any{
		"authenticated": true,
		"mode":          "web",
		"customer": map[string]any{
			"id":               customer.ID,
			"telegramId":       customer.TelegramID,
			"login":            customer.Login,
			"language":         customer.Language,
			"subscriptionLink": customer.SubscriptionLink,
			"expireAt":         customer.ExpireAt,
		},
		"purchase":       s.latestPurchasePayload(r.Context(), customer.ID),
		"purchases":      s.purchasesPayload(r.Context(), customer.ID),
		"trialAvailable": config.TrialDays() > 0 && customer.SubscriptionLink == nil,
	})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		Language string `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if request.Language == "" {
		request.Language = config.DefaultLanguage()
	}

	customer, sessionToken, err := s.authService.Register(r.Context(), request.Login, request.Password, request.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.setSessionCookie(w, r, sessionToken)
	writeJSON(w, map[string]any{
		"authenticated": true,
		"customer": map[string]any{
			"id":         customer.ID,
			"telegramId": customer.TelegramID,
			"login":      customer.Login,
			"language":   customer.Language,
		},
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	customer, sessionToken, err := s.authService.Login(r.Context(), request.Identifier, request.Password)
	if err != nil {
		if errors.Is(err, webauth.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.setSessionCookie(w, r, sessionToken)
	writeJSON(w, map[string]any{
		"authenticated": true,
		"customer": map[string]any{
			"id":         customer.ID,
			"telegramId": customer.TelegramID,
			"login":      customer.Login,
			"language":   customer.Language,
		},
	})
}

func (s *Server) linkTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	customer, err := s.currentCustomer(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if s.telegramLinkService == nil {
		http.Error(w, "telegram link service is unavailable", http.StatusServiceUnavailable)
		return
	}

	result, err := s.telegramLinkService.CreateLink(r.Context(), customer.ID)
	if err != nil {
		switch {
		case errors.Is(err, telegramlink.ErrBotURLNotConfigured):
			http.Error(w, "telegram bot url is not configured", http.StatusServiceUnavailable)
			return
		case errors.Is(err, database.ErrCustomerAlreadyLinked):
			http.Error(w, "telegram already linked", http.StatusConflict)
			return
		case errors.Is(err, database.ErrTelegramMergeNotAllowed):
			http.Error(w, "telegram merge is not allowed", http.StatusUnprocessableEntity)
			return
		case errors.Is(err, database.ErrTelegramLinkCustomerAbsent):
			http.Error(w, "customer not found", http.StatusNotFound)
			return
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	writeJSON(w, map[string]any{
		"url":       result.URL,
		"expiresAt": result.ExpiresAt,
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, map[string]any{
		"authenticated": false,
	})
}

func (s *Server) checkoutYookassa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	customer, err := s.currentCustomer(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Months int `json:"months"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if request.Months != 1 && request.Months != 3 && request.Months != 6 && request.Months != 12 {
		http.Error(w, "invalid months", http.StatusBadRequest)
		return
	}

	price := config.Price(request.Months)
	if price <= 0 {
		http.Error(w, "plan unavailable", http.StatusBadRequest)
		return
	}

	username := ""
	if customer.Login != nil {
		username = *customer.Login
	}
	ctx := context.WithValue(r.Context(), utils.ContextKeyUsername, username)
	ctx = context.WithValue(ctx, utils.ContextKeyReturnURL, fmt.Sprintf("%s/payment/return", config.FrontendOrigin()))

	paymentURL, purchaseID, err := s.paymentService.CreatePurchase(ctx, float64(price), request.Months, customer, database.InvoiceTypeYookasa)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"url":        paymentURL,
		"purchaseId": purchaseID,
	})
}

func (s *Server) purchaseStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	customer, err := s.currentCustomer(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	purchaseIDRaw := r.URL.Query().Get("id")
	if purchaseIDRaw == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	purchaseID, err := strconv.ParseInt(purchaseIDRaw, 10, 64)
	if err != nil || purchaseID <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	purchase, err := s.paymentServicePurchaseByID(r.Context(), purchaseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if purchase == nil {
		http.Error(w, "purchase not found", http.StatusNotFound)
		return
	}
	if purchase.CustomerID != customer.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := s.syncYookassaPurchase(r.Context(), purchase); err == nil {
		purchase, err = s.paymentServicePurchaseByID(r.Context(), purchaseID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if purchase == nil {
			http.Error(w, "purchase not found", http.StatusNotFound)
			return
		}
	}

	freshCustomer, err := s.customerRepository.FindById(r.Context(), customer.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"id":               purchase.ID,
		"status":           purchase.Status,
		"invoiceType":      purchase.InvoiceType,
		"paidAt":           purchase.PaidAt,
		"subscriptionLink": freshCustomer.SubscriptionLink,
		"expireAt":         freshCustomer.ExpireAt,
	})
}

func (s *Server) activateTrial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	customer, err := s.currentCustomer(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	subscriptionLink, err := s.paymentService.ActivateTrialForCustomer(r.Context(), customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	freshCustomer, err := s.customerRepository.FindById(r.Context(), customer.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"subscriptionLink": subscriptionLink,
		"expireAt":         freshCustomer.ExpireAt,
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := config.FrontendOrigin()
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf("encode response: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) currentCustomer(r *http.Request) (*database.Customer, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, webauth.ErrInvalidCredentials
	}

	return s.authService.ResolveSession(r.Context(), cookie.Value)
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPSRequest(r),
	})
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	if proto := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
		return true
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err == nil {
		return host != "localhost" && host != "127.0.0.1"
	}

	return !strings.HasPrefix(r.Host, "localhost") && !strings.HasPrefix(r.Host, "127.0.0.1")
}

func (s *Server) paymentServicePurchaseByID(ctx context.Context, purchaseID int64) (*database.Purchase, error) {
	return s.purchaseRepository.FindById(ctx, purchaseID)
}

func (s *Server) latestPurchasePayload(ctx context.Context, customerID int64) map[string]any {
	purchase, err := s.purchaseRepository.FindLatestByCustomer(ctx, customerID)
	if err != nil || purchase == nil {
		return nil
	}

	return s.purchasePayload(purchase)
}

func (s *Server) purchasesPayload(ctx context.Context, customerID int64) []map[string]any {
	purchases, err := s.purchaseRepository.FindByCustomer(ctx, customerID)
	if err != nil || purchases == nil {
		return []map[string]any{}
	}

	payload := make([]map[string]any, 0, len(*purchases))
	for i := range *purchases {
		payload = append(payload, s.purchasePayload(&(*purchases)[i]))
	}

	return payload
}

func (s *Server) purchasePayload(purchase *database.Purchase) map[string]any {
	payload := map[string]any{
		"id":          purchase.ID,
		"status":      purchase.Status,
		"invoiceType": purchase.InvoiceType,
		"amount":      purchase.Amount,
		"currency":    purchase.Currency,
		"months":      purchase.Month,
		"createdAt":   purchase.CreatedAt,
		"paidAt":      purchase.PaidAt,
	}

	if purchase.YookasaURL != nil {
		payload["paymentUrl"] = *purchase.YookasaURL
	}

	return payload
}

func (s *Server) syncCustomerPendingYookassaPurchases(ctx context.Context, customer *database.Customer) error {
	if customer == nil {
		return errors.New("customer is nil")
	}

	purchases, err := s.purchaseRepository.FindByCustomer(ctx, customer.ID)
	if err != nil || purchases == nil {
		return err
	}

	for i := range *purchases {
		purchase := (*purchases)[i]
		if err := s.syncYookassaPurchase(ctx, &purchase); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) syncYookassaPurchase(ctx context.Context, purchase *database.Purchase) error {
	if purchase == nil {
		return errors.New("purchase is nil")
	}

	if s.yookasaClient == nil ||
		purchase.InvoiceType != database.InvoiceTypeYookasa ||
		purchase.Status != database.PurchaseStatusPending ||
		purchase.YookasaID == nil {
		return nil
	}

	invoice, err := s.yookasaClient.GetPayment(ctx, *purchase.YookasaID)
	if err != nil {
		return err
	}

	if invoice.IsCancelled() {
		return s.paymentService.CancelYookassaPayment(purchase.ID)
	}

	if !invoice.Paid {
		return nil
	}

	ctxWithUsername := context.WithValue(ctx, utils.ContextKeyUsername, invoice.Metadata["username"])
	return s.paymentService.ProcessPurchaseById(ctxWithUsername, purchase.ID)
}
