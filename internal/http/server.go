package http

import (
	"fmt"
	"net/http"
	"remnawave-tg-shop-bot/internal/broadcast"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/http/handler"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/promo"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/sync"
	"remnawave-tg-shop-bot/internal/tribute"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

func NewServer(sh *handler.StatsHandler, pool *pgxpool.Pool, remnawaveClient *remnawave.Client, paymentService *payment.PaymentService, broadcastService *broadcast.Service, promoService *promo.Service, customerRepository *database.CustomerRepository, purchaseRepository *database.PurchaseRepository, referralRepository *database.ReferralRepository, syncService *sync.SyncService) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/healthcheck", handler.FullHealthHandler(pool, remnawaveClient))

	// Config endpoint (public)
	mux.HandleFunc("/api/config", handler.GetBotConfig)

	// Languages endpoint
	langHandler := handler.NewLanguagesHandler(customerRepository)
	mux.HandleFunc("/api/languages", langHandler.GetLanguages)

	// Auth handlers
	authHandler := handler.NewAuthHandler()
	mux.HandleFunc("/api/auth/check-admin", authHandler.CheckAdmin)

	// Sync endpoint
	syncHandler := handler.NewSyncHandler(syncService)
	mux.HandleFunc("/api/sync", authHandler.RequireAdmin(syncHandler.Sync))

	// Protected admin endpoints - require admin authentication
	mux.HandleFunc("/api/stats/totals", authHandler.RequireAdmin(sh.GetStatsTotals))
	mux.HandleFunc("/api/stats/growth", authHandler.RequireAdmin(sh.GetMonthlyGrowth))
	mux.HandleFunc("/api/stats/overview", authHandler.RequireAdmin(sh.GetStatsOverview))
	mux.HandleFunc("/api/stats/users/daily", authHandler.RequireAdmin(sh.GetDailyUserGrowth))
	mux.HandleFunc("/api/stats/revenue/daily", authHandler.RequireAdmin(sh.GetDailyRevenue))

	mux.HandleFunc("/api/users/stats/growth", authHandler.RequireAdmin(sh.GetUserGrowthStats))
	mux.HandleFunc("/api/users/{telegramID}", authHandler.RequireAdmin(sh.GetUserByTelegramID))

	// User management endpoints - all require admin privileges
	usersHandler := handler.NewUsersHandler(customerRepository, purchaseRepository, referralRepository)
	mux.HandleFunc("/api/users/search", authHandler.RequireAdmin(usersHandler.SearchUsers))
	mux.HandleFunc("/api/users/{telegramID}/payments", authHandler.RequireAdmin(usersHandler.GetUserPayments))
	mux.HandleFunc("/api/users/{telegramID}/update", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			usersHandler.UpdateUser(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/users/{telegramID}/delete", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			usersHandler.DeleteUser(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/users/{telegramID}/block", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			usersHandler.BlockUser(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/users/{telegramID}/unblock", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			usersHandler.UnblockUser(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	// Broadcast endpoints - require admin privileges
	brH := handler.NewBroadcastHandler(broadcastService)
	mux.HandleFunc("/api/broadcasts", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			brH.List(w, r)
		case http.MethodPost:
			brH.Create(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/broadcasts/{id}", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			brH.Delete(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	// Promo endpoints - require admin privileges
	promoH := handler.NewPromoHandler(promoService)
	mux.HandleFunc("/api/promos", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			promoH.List(w, r)
		case http.MethodPost:
			promoH.Create(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/promos/{id}", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			promoH.Update(w, r)
		case http.MethodDelete:
			promoH.Delete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	buildPath := "./tg-admin/dist/"
	fs := http.FileServer(http.Dir(buildPath))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// If it's an API request, let other handlers handle it
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// If path contains tgWebAppData or starts with /, serve index.html
		if strings.Contains(path, "tgWebAppData") || path == "/" {
			http.ServeFile(w, r, buildPath+"index.html")
			return
		}

		// Check if it's a static asset (has file extension)
		if strings.Contains(path, ".") {
			fs.ServeHTTP(w, r)
			return
		}

		// For all other routes (React Router routes), serve index.html
		http.ServeFile(w, r, buildPath+"index.html")
	})

	if config.GetTributeWebHookUrl() != "" {
		tributeHandler := tribute.NewClient(paymentService, customerRepository)
		mux.Handle(config.GetTributeWebHookUrl(), tributeHandler.WebHookHandler())
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", config.GetHealthCheckPort()),
		Handler: mux,
	}
}
