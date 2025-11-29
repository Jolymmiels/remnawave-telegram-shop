package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/promo"
	"remnawave-tg-shop-bot/internal/translation"
)

type PromoHandler struct {
	customerRepository *database.CustomerRepository
	promo              *promo.Service
	paymentService     *payment.PaymentService
	translation        *translation.Manager
}

func NewPromoHandler(
	customerRepository *database.CustomerRepository,
	promo *promo.Service,
	paymentService *payment.PaymentService,
	translation *translation.Manager,
) *PromoHandler {
	return &PromoHandler{
		customerRepository: customerRepository,
		promo:              promo,
		paymentService:     paymentService,
		translation:        translation,
	}
}

func (h *PromoHandler) PromoCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	validation, err := h.promo.ValidatePromoCode(ctx, promoCode, customer.ID)
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

func (h *PromoHandler) PromoCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	err = h.promo.ApplyPromoCode(ctx, promoID, customer.ID)
	if err != nil {
		slog.Error("Error applying promo code", "error", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_applying_promo"),
			ShowAlert:       true,
		})
		return
	}

	promoInfo, err := h.promo.GetPromoByID(ctx, promoID)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            h.translation.GetText(customer.Language, "error_applying_promo"),
			ShowAlert:       true,
		})
		return
	}

	err = h.paymentService.ExtendSubscription(ctx, customer.ID, promoInfo.BonusDays)
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
		Text:            fmt.Sprintf(h.translation.GetText(customer.Language, "promo_applied_success"), promoInfo.BonusDays),
		ShowAlert:       true,
	})

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      fmt.Sprintf(h.translation.GetText(customer.Language, "promo_applied_success"), promoInfo.BonusDays),
	})
}
