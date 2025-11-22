package handler

import (
	"context"
	"fmt"
	"net/http"
	"remnawave-tg-shop-bot/internal/remnawave"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func FullHealthHandler(pool *pgxpool.Pool, rw *remnawave.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{
			"status": "ok",
			"db":     "ok",
			"rw":     "ok",
			"time":   time.Now().Format(time.RFC3339),
		}

		dbCtx, dbCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer dbCancel()
		if err := pool.Ping(dbCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "fail"
			status["db"] = "error: " + err.Error()
		}

		rwCtx, rwCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer rwCancel()
		if err := rw.Ping(rwCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "fail"
			status["rw"] = "error: " + err.Error()
		}

		if status["status"] == "ok" {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"%s","db":"%s","remnawave":"%s","time":"%s"}`,
			status["status"], status["db"], status["rw"], status["time"])
	})
}
