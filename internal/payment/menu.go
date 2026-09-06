package payment

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"remnawave-tg-shop-bot/internal/database"
)

// Keep the navigation message, its photo and pin after payment. Only replace
// the payment buttons; this works for both photo and legacy text menus.
func (s PaymentService) refreshPurchaseMenu(ctx context.Context, purchaseID int64, customer *database.Customer) {
	messageID, ok := s.cache.Get(purchaseID)
	if !ok {
		return
	}
	_, err := s.telegramBot.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID: customer.TelegramID, MessageID: messageID,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: s.createConnectKeyboard(customer)},
	})
	if err != nil {
		slog.Warn("Failed to update menu after payment", "error", err)
	}
}
