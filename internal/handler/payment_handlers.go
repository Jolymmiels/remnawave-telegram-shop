package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
)

func (h Handler) BuyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode

	// Get active plans from database
	plans, err := h.planRepository.FindActive(ctx)
	if err != nil {
		slog.Error("Error fetching plans", "error", err)
		return
	}

	if len(plans) == 0 {
		slog.Error("No active plans found")
		return
	}

	var keyboard [][]models.InlineKeyboardButton

	// If there's only one plan, go directly to period selection
	if len(plans) == 1 {
		keyboard = h.buildPeriodSelectionKeyboard(plans[0], langCode)
	} else {
		// Show plan selection
		for _, plan := range plans {
			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{Text: plan.Name, CallbackData: fmt.Sprintf("%s?planId=%d", CallbackPlan, plan.ID)},
			})
		}
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
	})

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
		Text: h.translation.GetText(langCode, "pricing_info"),
	})

	if err != nil {
		slog.Error("Error sending buy message", "error", err)
	}
}

// PlanCallbackHandler handles plan selection when multiple plans exist
func (h Handler) PlanCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	langCode := update.CallbackQuery.From.LanguageCode

	planIdStr := callbackQuery["planId"]
	planId, err := strconv.ParseInt(planIdStr, 10, 64)
	if err != nil {
		slog.Error("Error parsing plan ID", "error", err)
		return
	}

	plan, err := h.planRepository.FindById(ctx, planId)
	if err != nil || plan == nil {
		slog.Error("Error fetching plan", "error", err, "planId", planId)
		return
	}

	keyboard := h.buildPeriodSelectionKeyboard(*plan, langCode)
	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackBuy},
	})

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
		Text: fmt.Sprintf("%s\n\n<b>%s</b>", h.translation.GetText(langCode, "pricing_info"), plan.Name),
	})

	if err != nil {
		slog.Error("Error sending plan message", "error", err)
	}
}

func (h Handler) buildPeriodSelectionKeyboard(plan database.Plan, langCode string) [][]models.InlineKeyboardButton {
	var priceButtons []models.InlineKeyboardButton

	if plan.Price1 > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_1"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, 1, plan.Price1, plan.ID),
		})
	}

	if plan.Price3 > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_3"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, 3, plan.Price3, plan.ID),
		})
	}

	if plan.Price6 > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_6"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, 6, plan.Price6, plan.ID),
		})
	}

	if plan.Price12 > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_12"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, 12, plan.Price12, plan.ID),
		})
	}

	var keyboard [][]models.InlineKeyboardButton
	if len(priceButtons) == 4 {
		keyboard = append(keyboard, priceButtons[:2])
		keyboard = append(keyboard, priceButtons[2:])
	} else if len(priceButtons) > 0 {
		keyboard = append(keyboard, priceButtons)
	}

	return keyboard
}

func (h Handler) SellCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	langCode := update.CallbackQuery.From.LanguageCode
	month := callbackQuery["month"]
	amount := callbackQuery["amount"]
	planId := callbackQuery["planId"]

	var keyboard [][]models.InlineKeyboardButton

	if config.IsCryptoPayEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "crypto_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s&planId=%s", CallbackPayment, month, database.InvoiceTypeCrypto, amount, planId)},
		})
	}

	if config.IsYookasaEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "card_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s&planId=%s", CallbackPayment, month, database.InvoiceTypeYookasa, amount, planId)},
		})
	}

	if config.IsTelegramStarsEnabled() {
		shouldShowStarsButton := true

		if config.RequirePaidPurchaseForStars() {
			customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
			if err != nil {
				slog.Error("Error finding customer for stars check", "error", err)
				shouldShowStarsButton = false
			} else if customer != nil {
				paidPurchase, err := h.purchaseRepository.FindSuccessfulPaidPurchaseByCustomer(ctx, customer.ID)
				if err != nil {
					slog.Error("Error checking paid purchase", "error", err)
					shouldShowStarsButton = false
				} else if paidPurchase == nil {
					shouldShowStarsButton = false
				}
			} else {
				shouldShowStarsButton = false
			}
		}

		if shouldShowStarsButton {
			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{Text: h.translation.GetText(langCode, "stars_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s&planId=%s", CallbackPayment, month, database.InvoiceTypeTelegram, amount, planId)},
			})
		}
	}

	if config.GetTributeWebHookUrl() != "" {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "tribute_button"), URL: config.GetTributePaymentUrl()},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackBuy},
	})

	_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	if err != nil {
		slog.Error("Error sending sell message", "error", err)
	}
}

func (h Handler) PaymentCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	month, err := strconv.Atoi(callbackQuery["month"])
	if err != nil {
		slog.Error("Error getting month from query", "error", err)
		return
	}

	invoiceType := database.InvoiceType(callbackQuery["invoiceType"])
	amountStr := callbackQuery["amount"]
	planIdStr := callbackQuery["planId"]

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		slog.Error("Error parsing amount", "error", err)
		return
	}

	planId, err := strconv.ParseInt(planIdStr, 10, 64)
	if err != nil {
		slog.Error("Error parsing plan ID", "error", err)
		return
	}

	plan, err := h.planRepository.FindById(ctx, planId)
	if err != nil || plan == nil {
		slog.Error("Error fetching plan", "error", err, "planId", planId)
		return
	}

	var price int
	if invoiceType == database.InvoiceTypeTelegram {
		// Calculate stars price using exchange rate from settings
		exchangeRate := h.settingsRepository.GetFloat("stars_exchange_rate", 1.5)
		price = plan.GetStarsPrice(month, exchangeRate)
	} else {
		price = amount
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	ctxWithUsername := context.WithValue(ctx, "username", update.CallbackQuery.From.Username)
	paymentURL, purchaseId, err := h.paymentService.CreatePurchase(ctxWithUsername, float64(price), month, customer, invoiceType)
	if err != nil {
		slog.Error("Error creating payment", "error", err)
		return
	}

	langCode := update.CallbackQuery.From.LanguageCode

	message, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(langCode, "pay_button"), URL: paymentURL},
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, month, amount, planId)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error updating sell message", "error", err)
		return
	}
	h.cache.Set(purchaseId, message.ID)
}

func (h Handler) PreCheckoutCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.AnswerPreCheckoutQuery(ctx, &bot.AnswerPreCheckoutQueryParams{
		PreCheckoutQueryID: update.PreCheckoutQuery.ID,
		OK:                 true,
	})
	if err != nil {
		slog.Error("Error sending answer pre checkout query", "error", err)
	}
}

func (h Handler) SuccessPaymentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	payload := strings.Split(update.Message.SuccessfulPayment.InvoicePayload, "&")
	purchaseId, err := strconv.Atoi(payload[0])
	username := payload[1]
	if err != nil {
		slog.Error("Error parsing purchase id", "error", err)
		return
	}

	ctxWithUsername := context.WithValue(ctx, "username", username)
	err = h.paymentService.ProcessPurchaseById(ctxWithUsername, int64(purchaseId))
	if err != nil {
		slog.Error("Error processing purchase", "error", err)
	}
}

func parseCallbackData(data string) map[string]string {
	result := make(map[string]string)

	parts := strings.Split(data, "?")
	if len(parts) < 2 {
		return result
	}

	params := strings.Split(parts[1], "&")
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}

	return result
}
