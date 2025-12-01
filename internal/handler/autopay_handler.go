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

	// Refresh the payment methods view if coming from payment methods screen
	callback := update.CallbackQuery.Message.Message
	customer.AutopayEnabled = false
	text := h.buildPaymentMethodsText(customer, langCode)
	markup := h.buildPaymentMethodsKeyboard(customer, langCode)

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		Text:      text,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: markup,
		},
	})
}

func (h *AutopayHandler) PaymentMethodsCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery.Message.Message
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

	text := h.buildPaymentMethodsText(customer, langCode)
	markup := h.buildPaymentMethodsKeyboard(customer, langCode)

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		Text:      text,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: markup,
		},
	})
	if err != nil {
		slog.Error("Error sending payment methods message", "error", err)
	}
}

func (h *AutopayHandler) AutopayEnableCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	if customer.PaymentMethodID == nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "no_payment_method"),
			ShowAlert:       true,
		})
		return
	}

	err = h.paymentService.EnableAutopay(ctx, telegramID)
	if err != nil {
		slog.Error("Error enabling autopay", "error", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "autopay_enable_error"),
			ShowAlert:       true,
		})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            h.translation.GetText(langCode, "autopay_enabled"),
		ShowAlert:       true,
	})

	// Refresh the payment methods view
	customer.AutopayEnabled = true
	callback := update.CallbackQuery.Message.Message
	text := h.buildPaymentMethodsText(customer, langCode)
	markup := h.buildPaymentMethodsKeyboard(customer, langCode)

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		Text:      text,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: markup,
		},
	})
}

func (h *AutopayHandler) DeletePaymentMethodCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	if customer.PaymentMethodID == nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "no_payment_method"),
			ShowAlert:       true,
		})
		return
	}

	err = h.paymentService.DeletePaymentMethod(ctx, telegramID)
	if err != nil {
		slog.Error("Error deleting payment method", "error", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(langCode, "payment_method_delete_error"),
			ShowAlert:       true,
		})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            h.translation.GetText(langCode, "payment_method_deleted"),
		ShowAlert:       true,
	})

	// Refresh the payment methods view
	customer.PaymentMethodID = nil
	customer.AutopayEnabled = false
	callback := update.CallbackQuery.Message.Message
	text := h.buildPaymentMethodsText(customer, langCode)
	markup := h.buildPaymentMethodsKeyboard(customer, langCode)

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		Text:      text,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: markup,
		},
	})
}

func (h *AutopayHandler) buildPaymentMethodsText(customer *database.Customer, langCode string) string {
	text := h.translation.GetText(langCode, "payment_methods_header") + "\n\n"

	if customer.PaymentMethodID != nil {
		maskedID := utils.MaskHalf(*customer.PaymentMethodID)
		text += h.translation.GetText(langCode, "saved_payment_method") + ": " + maskedID + "\n"

		if customer.AutopayEnabled {
			text += h.translation.GetText(langCode, "autopay_status") + ": " + h.translation.GetText(langCode, "autopay_on") + " ✅\n"
		} else {
			text += h.translation.GetText(langCode, "autopay_status") + ": " + h.translation.GetText(langCode, "autopay_off") + " ❌\n"
		}
	} else {
		text += h.translation.GetText(langCode, "no_saved_payment_methods") + "\n"
	}

	return text
}

func (h *AutopayHandler) buildPaymentMethodsKeyboard(customer *database.Customer, langCode string) [][]models.InlineKeyboardButton {
	var keyboard [][]models.InlineKeyboardButton

	if customer.PaymentMethodID != nil {
		if customer.AutopayEnabled {
			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{Text: h.translation.GetText(langCode, "disable_autopay_button"), CallbackData: CallbackAutopayDisable},
			})
		} else {
			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{Text: h.translation.GetText(langCode, "enable_autopay_button"), CallbackData: CallbackAutopayEnable},
			})
		}
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "delete_payment_method_button"), CallbackData: CallbackDeletePayment},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackConnect},
	})

	return keyboard
}
