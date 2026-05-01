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
	"remnawave-tg-shop-bot/internal/remnawave"
)

func (h Handler) BuyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode

	var priceButtons []models.InlineKeyboardButton

	if config.Price1() > 0 {
		priceButtons = append(priceButtons, h.translation.GetButton(langCode, "month_1").InlineCallback(fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 1, config.Price1())))
	}

	if config.Price3() > 0 {
		priceButtons = append(priceButtons, h.translation.GetButton(langCode, "month_3").InlineCallback(fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 3, config.Price3())))
	}

	if config.Price6() > 0 {
		priceButtons = append(priceButtons, h.translation.GetButton(langCode, "month_6").InlineCallback(fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 6, config.Price6())))
	}

	if config.Price12() > 0 {
		priceButtons = append(priceButtons, h.translation.GetButton(langCode, "month_12").InlineCallback(fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 12, config.Price12())))
	}

	keyboard := [][]models.InlineKeyboardButton{}

	if len(priceButtons) == 4 {
		keyboard = append(keyboard, priceButtons[:2])
		keyboard = append(keyboard, priceButtons[2:])
	} else if len(priceButtons) > 0 {
		keyboard = append(keyboard, priceButtons)
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		h.translation.GetButton(langCode, "back_button").InlineCallback(CallbackStart),
	})

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
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

func (h Handler) SellCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	langCode := update.CallbackQuery.From.LanguageCode
	month := callbackQuery["month"]
	amount := callbackQuery["amount"]

	var keyboard [][]models.InlineKeyboardButton

	if config.IsCryptoPayEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "crypto_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeCrypto, amount)),
		})
	}

	if config.IsYookasaEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "card_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeYookasa, amount)),
		})
	}

	if config.IsPlategaSBPEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "platega_sbp_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypePlategaSBP, amount)),
		})
	}

	if config.IsPlategaCardsEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "platega_cards_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypePlategaCards, amount)),
		})
	}

	if config.IsPlategaAcquiringEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "platega_acquiring_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypePlategaAcquiring, amount)),
		})
	}

	if config.IsPlategaWorldwideEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "platega_worldwide_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypePlategaWorldwide, amount)),
		})
	}

	if config.IsPlategaCryptoEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "platega_crypto_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypePlategaCrypto, amount)),
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
				h.translation.GetButton(langCode, "stars_button").InlineCallback(fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeTelegram, amount)),
			})
		}
	}

	if config.GetTributeWebHookUrl() != "" {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			h.translation.GetButton(langCode, "tribute_button").InlineURL(config.GetTributePaymentUrl()),
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		h.translation.GetButton(langCode, "back_button").InlineCallback(CallbackBuy),
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

	var price int
	if invoiceType == database.InvoiceTypeTelegram {
		price = config.StarsPrice(month)
	} else {
		price = config.Price(month)
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

	ctxWithUsername := context.WithValue(ctx, remnawave.CtxKeyUsername, update.CallbackQuery.From.Username)
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
					h.translation.GetButton(langCode, "pay_button").InlineURL(paymentURL),
					h.translation.GetButton(langCode, "back_button").InlineCallback(fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, month, price)),
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

	ctxWithUsername := context.WithValue(ctx, remnawave.CtxKeyUsername, username)
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
