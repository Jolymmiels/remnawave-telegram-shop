package handler

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"strconv"
	"strings"
)

func (h *Handler) PromoCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	customer, err := h.customerRepository.FindByTelegramId(ctx, update.Message.From.ID)
	if err != nil || customer == nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   h.translation.GetText("en", "error_customer_not_found"),
		})
		return
	}

	messageParts := strings.Fields(update.Message.Text)
	if len(messageParts) < 2 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   h.translation.GetText(customer.Language, "promo_usage"),
		})
		return
	}

	promoCode := strings.TrimSpace(messageParts[1])

	// Validate promo code
	validation, err := h.promoService.ValidatePromoCode(ctx, promoCode, customer.ID)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   h.translation.GetText(customer.Language, "error_validating_promo"),
		})
		return
	}

	if !validation.Valid {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   validation.Message,
		})
		return
	}

	// Show promo confirmation
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         h.translation.GetText(customer.Language, "apply_promo"),
					CallbackData: fmt.Sprintf("%s_%d", CallbackPromo, validation.PromoID),
				},
			},
			{
				{
					Text:         h.translation.GetText(customer.Language, "cancel"),
					CallbackData: CallbackStart,
				},
			},
		},
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        validation.Message,
		ReplyMarkup: keyboard,
	})
}

func (h *Handler) PromoCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	customer, err := h.customerRepository.FindByTelegramId(ctx, update.CallbackQuery.From.ID)
	if err != nil || customer == nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText("en", "error_customer_not_found"),
			ShowAlert:       true,
		})
		return
	}

	// Extract promo ID from callback data
	parts := strings.Split(update.CallbackQuery.Data, "_")
	if len(parts) != 2 {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_invalid_promo"),
			ShowAlert:       true,
		})
		return
	}

	promoID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_invalid_promo"),
			ShowAlert:       true,
		})
		return
	}

	// Apply the promo code
	err = h.promoService.ApplyPromoCode(ctx, promoID, customer.ID)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_applying_promo"),
			ShowAlert:       true,
		})
		return
	}

	// Update customer expiry date with bonus days
	promo, err := h.promoService.GetPromoByID(ctx, promoID)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_applying_promo"),
			ShowAlert:       true,
		})
		return
	}

	// Extend subscription with bonus days
	err = h.paymentService.ExtendSubscription(ctx, customer.ID, promo.BonusDays)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_applying_promo"),
			ShowAlert:       true,
		})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            fmt.Sprintf(h.translation.GetText(customer.Language, "promo_applied_success"), promo.BonusDays),
		ShowAlert:       true,
	})

	// Edit message to show success
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      fmt.Sprintf(h.translation.GetText(customer.Language, "promo_applied_success"), promo.BonusDays),
	})
}
