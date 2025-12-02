package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/translation"
)

type PaymentHandler struct {
	customerRepository *database.CustomerRepository
	purchaseRepository *database.PurchaseRepository
	planRepository     *database.PlanRepository
	settingsRepository *database.SettingsRepository
	paymentService     *payment.PaymentService
	translation        *translation.Manager
	cache              *cache.Cache
}

func NewPaymentHandler(
	customerRepository *database.CustomerRepository,
	purchaseRepository *database.PurchaseRepository,
	planRepository *database.PlanRepository,
	settingsRepository *database.SettingsRepository,
	paymentService *payment.PaymentService,
	translation *translation.Manager,
	cache *cache.Cache,
) *PaymentHandler {
	return &PaymentHandler{
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		planRepository:     planRepository,
		settingsRepository: settingsRepository,
		paymentService:     paymentService,
		translation:        translation,
		cache:              cache,
	}
}

func (h *PaymentHandler) BuyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode

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

	if len(plans) == 1 {
		keyboard = h.buildPeriodSelectionKeyboard(plans[0], langCode)
	} else {
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

func (h *PaymentHandler) PlanCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

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

func (h *PaymentHandler) buildPeriodSelectionKeyboard(plan database.Plan, langCode string) [][]models.InlineKeyboardButton {
	var priceButtons []models.InlineKeyboardButton
	showPrice := config.PeriodButtonsShowPrice()

	formatButton := func(monthKey string, price int, months int) models.InlineKeyboardButton {
		var text string
		if showPrice {
			text = fmt.Sprintf("%s - %d ₽", h.translation.GetText(langCode, monthKey), price)
		} else {
			text = h.translation.GetText(langCode, monthKey)
		}
		return models.InlineKeyboardButton{
			Text:         text,
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d&planId=%d", CallbackSell, months, price, plan.ID),
		}
	}

	if plan.Price1 > 0 {
		priceButtons = append(priceButtons, formatButton("month_1", plan.Price1, 1))
	}

	if plan.Price3 > 0 {
		priceButtons = append(priceButtons, formatButton("month_3", plan.Price3, 3))
	}

	if plan.Price6 > 0 {
		priceButtons = append(priceButtons, formatButton("month_6", plan.Price6, 6))
	}

	if plan.Price12 > 0 {
		priceButtons = append(priceButtons, formatButton("month_12", plan.Price12, 12))
	}

	var keyboard [][]models.InlineKeyboardButton
	layout := config.PeriodButtonsLayout()

	switch layout {
	case "1x4":
		keyboard = append(keyboard, priceButtons)
	case "4x1":
		for _, btn := range priceButtons {
			keyboard = append(keyboard, []models.InlineKeyboardButton{btn})
		}
	case "2x2":
		for i := 0; i < len(priceButtons); i += 2 {
			if i+1 < len(priceButtons) {
				keyboard = append(keyboard, []models.InlineKeyboardButton{priceButtons[i], priceButtons[i+1]})
			} else {
				keyboard = append(keyboard, []models.InlineKeyboardButton{priceButtons[i]})
			}
		}
	default:
		keyboard = append(keyboard, priceButtons)
	}

	return keyboard
}

func (h *PaymentHandler) SellCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

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
			{Text: h.translation.GetText(langCode, "card_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s&planId=%s", CallbackPayment, month, database.InvoiceTypeYookassa, amount, planId)},
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

	if config.IsTributeEnabled() {
		planIdInt, _ := strconv.ParseInt(planId, 10, 64)
		plan, _ := h.planRepository.FindById(ctx, planIdInt)
		if plan != nil && plan.TributeURL != "" {
			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{Text: h.translation.GetText(langCode, "tribute_button"), URL: plan.TributeURL},
			})
		}
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

func (h *PaymentHandler) PaymentCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)

	monthStr := callbackQuery["month"]
	if monthStr == "" {
		slog.Warn("Empty month in payment callback, ignoring", "data", update.CallbackQuery.Data)
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		slog.Error("Error getting month from query", "error", err, "month", monthStr)
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
		exchangeRate := h.settingsRepository.GetFloat("stars_exchange_rate", 1.5)
		price = plan.GetStarsPrice(month, exchangeRate)
	} else {
		price = amount
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctxWithTimeout, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	ctxWithUsername := context.WithValue(ctxWithTimeout, "username", update.CallbackQuery.From.Username)
	paymentURL, purchaseId, err := h.paymentService.CreatePurchase(ctxWithUsername, float64(price), month, customer, invoiceType, &planId)
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

func (h *PaymentHandler) PreCheckoutCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.AnswerPreCheckoutQuery(ctx, &bot.AnswerPreCheckoutQueryParams{
		PreCheckoutQueryID: update.PreCheckoutQuery.ID,
		OK:                 true,
	})
	if err != nil {
		slog.Error("Error sending answer pre checkout query", "error", err)
	}
}

func (h *PaymentHandler) SuccessPaymentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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
