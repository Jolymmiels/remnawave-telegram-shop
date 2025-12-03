package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) runBroadcast(adminChatID int64, b *bot.Bot, text string, lang string) {
	ctx := context.Background()
	ids, err := h.customerRepository.GetAllTelegramIDs(ctx)
	if err != nil {
		slog.Error("Failed to get all Telegram IDs for broadcast", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    adminChatID,
			Text:      fmt.Sprintf(h.translation.GetText(lang, "broadcast_database_error"), err),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	slog.Info("Starting broadcast", "total_users", len(ids))
	sentCount := 0
	failedCount := 0
	for _, id := range ids {
		time.Sleep(100 * time.Millisecond)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    id,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			slog.Warn("Failed to send message to user", "user_id", id, "error", err)
			failedCount++
		} else {
			sentCount++
		}
	}
	reportMsg := fmt.Sprintf(
		h.translation.GetText(lang, "broadcast_report"),
		len(ids),
		sentCount,
		failedCount,
	)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    adminChatID,
		Text:      reportMsg,
		ParseMode: models.ParseModeHTML,
	})
	slog.Info("Broadcast finished", "sent", sentCount, "failed", failedCount)
}

func (h *Handler) BroadcastCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	slog.Info("Broadcast command invoked!")
	langCode := update.Message.From.LanguageCode
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	fullText := update.Message.Text
	var commandEndIndex int
	if len(update.Message.Entities) > 0 && update.Message.Entities[0].Type == "bot_command" {
		commandEndIndex = int(update.Message.Entities[0].Offset) + int(update.Message.Entities[0].Length)
	} else {
		parts := strings.SplitN(fullText, " ", 2)
		commandEndIndex = len(parts[0])
	}
	broadcastText := strings.TrimSpace(fullText[commandEndIndex:])
	if broadcastText == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      h.translation.GetText(langCode, "broadcast_empty_text_error"),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	startMsg := fmt.Sprintf(
		h.translation.GetText(langCode, "broadcast_started"),
		broadcastText,
	)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      startMsg,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Error("Error sending broadcast start message to admin", "error", err)
	}
	go h.runBroadcast(update.Message.Chat.ID, b, broadcastText, langCode)
}
