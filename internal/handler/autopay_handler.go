package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/utils"
)

type AutopayHandler struct {
	customerRepository *database.CustomerRepository
	paymentService     *payment.PaymentService
	translation        *translation.Manager
}

func NewAutopayHandler(
	customerRepository *database.CustomerRepository,
	paymentService *payment.PaymentService,
	translation *translation.Manager,
) *AutopayHandler {
	return &AutopayHandler{
		customerRepository: customerRepository,
		paymentService:     paymentService,
		translation:        translation,
	}
}

func (h *AutopayHandler) AutopayDisableCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	langCode := update.CallbackQuery.From.LanguageCode

	customer, err := h.customerRepository.FindByTelegramId(ctx, telegramID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if customer == nil {
		slog.Error("Customer not found", "telegramId", utils.MaskHalfInt64(telegramID))
		return
	}

	if !customer.AutopayEnabled || customer.PaymentMethodID == nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "autopay_not_enabled"),
			ShowAlert:       true,
		})
		return
	}

	err = h.paymentService.DisableAutopay(ctx, telegramID)
	if err != nil {
		slog.Error("Error disabling autopay", "error", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "autopay_disable_error"),
			ShowAlert:       true,
		})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            h.translation.GetText(langCode, "autopay_disabled"),
		ShowAlert:       true,
	})

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    telegramID,
		Text:      h.translation.GetText(langCode, "autopay_disabled_message"),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
			},
		},
	})
	if err != nil {
		slog.Error("Error sending autopay disabled message", "error", err)
	}
}
