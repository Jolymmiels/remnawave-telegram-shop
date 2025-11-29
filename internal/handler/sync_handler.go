package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/sync"
)

type SyncHandler struct {
	syncService *sync.SyncService
}

func NewSyncHandler(syncService *sync.SyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

func (h *SyncHandler) SyncUsersCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.syncService.Sync(ctx)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Users synced",
	})
	if err != nil {
		slog.Error("Error sending sync message", "error", err)
	}
}
