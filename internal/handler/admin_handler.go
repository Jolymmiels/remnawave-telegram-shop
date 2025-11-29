package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) AdminCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	webAppURL := config.BotAdminURL()
	if !strings.HasSuffix(webAppURL, "/") {
		webAppURL += "/"
	}

	msg := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Добро пожаловать в админку!",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "🔐 Админ панель", WebApp: &models.WebAppInfo{URL: webAppURL}},
				},
			},
		},
	}
	_, err := b.SendMessage(ctx, msg)
	if err != nil {
		slog.Error("Failed to send admin message", "error", err)
	}
}
