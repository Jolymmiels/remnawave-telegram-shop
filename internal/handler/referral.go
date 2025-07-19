package handler

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

func (h Handler) ReferralCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	customer, _ := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	langCode := update.CallbackQuery.From.LanguageCode
	refCode := customer.TelegramID

	refLink := fmt.Sprintf("https://telegram.me/share/url?url=https://t.me/%s?start=ref_%d", update.CallbackQuery.Message.Message.From.Username, refCode)
	count, err := h.referralRepository.CountByReferrer(ctx, customer.TelegramID)
	if err != nil {
		slog.Error("error counting referrals", err)
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
				{Text: h.translation.GetText(langCode, "create_promocode_button"), CallbackData: CallbackPromoCreate},
			},
			{
				{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
			},
		}},
	})
	if err != nil {
		slog.Error("Error sending referral message", err)
	}
}

func (h Handler) PromoCreateCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	customer, _ := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	lang := update.CallbackQuery.From.LanguageCode
	if customer == nil {
		return
	}
	code, err := h.paymentService.CreatePromocode(ctx, customer, 1, 1)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.CallbackQuery.Message.Message.Chat.ID, Text: h.translation.GetText(lang, "insufficient_balance")})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.CallbackQuery.Message.Message.Chat.ID, Text: fmt.Sprintf(h.translation.GetText(lang, "promocode_created"), code)})
}
