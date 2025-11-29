package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/utils"
)

type TrialHandler struct {
	customerRepository *database.CustomerRepository
	paymentService     *payment.PaymentService
	translation        *translation.Manager
}

func NewTrialHandler(
	customerRepository *database.CustomerRepository,
	paymentService *payment.PaymentService,
	translation *translation.Manager,
) *TrialHandler {
	return &TrialHandler{
		customerRepository: customerRepository,
		paymentService:     paymentService,
		translation:        translation,
	}
}

func (h *TrialHandler) TrialCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	if !config.IsTrialEnabled() {
		return
	}
	c, err := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if c == nil {
		slog.Error("customer not exist", "telegramId", utils.MaskHalfInt64(update.CallbackQuery.From.ID), "error", err)
		return
	}
	if c.SubscriptionLink != nil || c.TrialUsed {
		return
	}
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		Text:      h.translation.GetText(langCode, "trial_text"),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: h.translation.GetText(langCode, "activate_trial_button"), CallbackData: CallbackActivateTrial}},
			{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
		}},
	})
	if err != nil {
		slog.Error("Error sending /trial message", "error", err)
	}
}

func (h *TrialHandler) ActivateTrialCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	if !config.IsTrialEnabled() {
		return
	}
	c, err := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if c == nil {
		slog.Error("customer not exist", "telegramId", utils.MaskHalfInt64(update.CallbackQuery.From.ID), "error", err)
		return
	}
	if c.SubscriptionLink != nil || c.TrialUsed {
		return
	}
	callback := update.CallbackQuery.Message.Message
	ctxWithUsername := context.WithValue(ctx, "username", update.CallbackQuery.From.Username)
	_, err = h.paymentService.ActivateTrial(ctxWithUsername, update.CallbackQuery.From.ID)
	if err == nil {
		_ = h.customerRepository.UpdateFields(ctx, c.ID, map[string]interface{}{"trial_used": true})
	}
	langCode := update.CallbackQuery.From.LanguageCode
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      callback.Chat.ID,
		MessageID:   callback.ID,
		Text:        h.translation.GetText(langCode, "trial_activated"),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: h.createConnectKeyboard(langCode)},
	})
	if err != nil {
		slog.Error("Error sending /trial message", "error", err)
	}
}

func (h *TrialHandler) createConnectKeyboard(lang string) [][]models.InlineKeyboardButton {
	var inlineCustomerKeyboard [][]models.InlineKeyboardButton
	inlineCustomerKeyboard = append(inlineCustomerKeyboard, h.resolveConnectButton(lang))
	inlineCustomerKeyboard = append(inlineCustomerKeyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(lang, "back_button"), CallbackData: CallbackStart},
	})
	return inlineCustomerKeyboard
}

func (h *TrialHandler) resolveConnectButton(lang string) []models.InlineKeyboardButton {
	var inlineKeyboard []models.InlineKeyboardButton

	if config.GetMiniAppURL() != "" {
		inlineKeyboard = []models.InlineKeyboardButton{
			{Text: h.translation.GetText(lang, "connect_button"), WebApp: &models.WebAppInfo{
				URL: config.GetMiniAppURL(),
			}},
		}
	} else {
		inlineKeyboard = []models.InlineKeyboardButton{
			{Text: h.translation.GetText(lang, "connect_button"), CallbackData: CallbackConnect},
		}
	}
	return inlineKeyboard
}
