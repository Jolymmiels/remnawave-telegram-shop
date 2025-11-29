package handler

import (
	"context"
	"net/http"
	"remnawave-tg-shop-bot/internal/sync"
	"time"
)

type SyncHandler struct {
	syncService *sync.SyncService
}

func NewSyncHandler(syncService *sync.SyncService) *SyncHandler {
	return &SyncHandler{syncService: syncService}
}

func (h *SyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		h.syncService.Sync(ctx)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"sync started"}`))
}
