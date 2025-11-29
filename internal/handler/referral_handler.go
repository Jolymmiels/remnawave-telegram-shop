package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/translation"
)

type ReferralHandler struct {
	customerRepository *database.CustomerRepository
	referralRepository *database.ReferralRepository
	translation        *translation.Manager
}

func NewReferralHandler(
	customerRepository *database.CustomerRepository,
	referralRepository *database.ReferralRepository,
	translation *translation.Manager,
) *ReferralHandler {
	return &ReferralHandler{
		customerRepository: customerRepository,
		referralRepository: referralRepository,
		translation:        translation,
	}
}

func (h *ReferralHandler) ReferralCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	customer, _ := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	langCode := update.CallbackQuery.From.LanguageCode
	refCode := customer.TelegramID

	refLink := fmt.Sprintf("https://telegram.me/share/url?url=https://t.me/%s?start=ref_%d", update.CallbackQuery.Message.Message.From.Username, refCode)
	count, err := h.referralRepository.CountByReferrer(ctx, customer.TelegramID)
	if err != nil {
		slog.Error("error counting referrals", "error", err)
		return
	}
	text := fmt.Sprintf(h.translation.GetText(langCode, "referral_text"), count)
	callbackMessage := update.CallbackQuery.Message.Message
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callbackMessage.Chat.ID,
		MessageID: callbackMessage.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: h.translation.GetText(langCode, "share_referral_button"), URL: refLink},
			},
			{
				{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
			},
		}},
	})
	if err != nil {
		slog.Error("Error sending referral message", "error", err)
	}
}
