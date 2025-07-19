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
	"remnawave-tg-shop-bot/utils"
)

func (h Handler) BuyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode

	var priceButtons []models.InlineKeyboardButton

	if config.Price1() > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_1"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 1, config.Price1()),
		})
	}

	if config.Price3() > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_3"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 3, config.Price3()),
		})
	}

	if config.Price6() > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_6"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 6, config.Price6()),
		})
	}

	if config.Price12() > 0 {
		priceButtons = append(priceButtons, models.InlineKeyboardButton{
			Text:         h.translation.GetText(langCode, "month_12"),
			CallbackData: fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, 12, config.Price12()),
		})
	}

	keyboard := [][]models.InlineKeyboardButton{}

	if len(priceButtons) == 4 {
		keyboard = append(keyboard, priceButtons[:2])
		keyboard = append(keyboard, priceButtons[2:])
	} else if len(priceButtons) > 0 {
		keyboard = append(keyboard, priceButtons)
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
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
		slog.Error("Error sending buy message", err)
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
			{Text: h.translation.GetText(langCode, "crypto_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeCrypto, amount)},
		})
	}

	if config.IsYookasaEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "card_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeYookasa, amount)},
		})
	}

	if config.IsTelegramStarsEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "stars_button"), CallbackData: fmt.Sprintf("%s?month=%s&invoiceType=%s&amount=%s", CallbackPayment, month, database.InvoiceTypeTelegram, amount)},
		})
	}

	if config.GetTributeWebHookUrl() != "" {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "tribute_button"), URL: config.GetTributePaymentUrl()},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "buy_sub_balance_button"), CallbackData: fmt.Sprintf("%s?month=%s", CallbackPayFromBal, month)},
	})

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
		slog.Error("Error sending sell message", err)
	}
}

func (h Handler) PaymentCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	month, err := strconv.Atoi(callbackQuery["month"])
	if err != nil {
		slog.Error("Error getting month from query", err)
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
		slog.Error("Error finding customer", err)
		return
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	ctxWithUsername := context.WithValue(ctx, "username", update.CallbackQuery.From.Username)
	paymentURL, purchaseId, err := h.paymentService.CreatePurchase(ctxWithUsername, price, month, customer, invoiceType)
	if err != nil {
		slog.Error("Error creating payment", err)
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
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: fmt.Sprintf("%s?month=%d&amount=%d", CallbackSell, month, price)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error updating sell message", err)
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
		slog.Error("Error sending answer pre checkout query", err)
	}
}

func (h Handler) SuccessPaymentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	payload := strings.Split(update.Message.SuccessfulPayment.InvoicePayload, "&")
	purchaseId, err := strconv.Atoi(payload[0])
	username := payload[1]
	if err != nil {
		slog.Error("Error parsing purchase id", err)
		return
	}

	ctxWithUsername := context.WithValue(ctx, "username", username)
	err = h.paymentService.ProcessPurchaseById(ctxWithUsername, int64(purchaseId))
	if err != nil {
		slog.Error("Error processing purchase", err)
	}
}

func (h Handler) BalanceCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	lang := update.CallbackQuery.From.LanguageCode
	customer, _ := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if customer == nil {
		return
	}

	user, _ := h.paymentService.GetUser(ctx, customer.TelegramID)
	var info strings.Builder
	if user != nil {
		expire := user.ExpireAt.Format("02.01.2006 15:04")
		status := "ACTIVE"
		if user.Status.Set {
			status = string(user.Status.Value)
		}
		lastClient := "-"
		if !user.LastConnectedNode.Null {
			lastClient = user.LastConnectedNode.Value.GetNodeName()
		}
		start := time.Now().Truncate(24 * time.Hour)
		usage, _ := h.paymentService.GetUserDailyUsage(ctx, user.UUID.String(), start, time.Now())
		limit := 0.0
		if v, ok := user.TrafficLimitBytes.Get(); ok {
			limit = float64(v)
		}
		info.WriteString("📰 Информация об аккаунте:\n\n")
		info.WriteString(fmt.Sprintf("├ Баланс: <%.0f ₽>\n", customer.Balance))
		info.WriteString(fmt.Sprintf("├ Подписка до: <%s>\n", expire))
		info.WriteString(fmt.Sprintf("├ Статус: <%s>\n", status))
		info.WriteString(fmt.Sprintf("├ Последний клиент: <%s>\n\n", lastClient))
		info.WriteString("🌐 Информация о трафике:\n\n")
		info.WriteString(fmt.Sprintf("├ Лимит в сутки: <%s / %s>\n", utils.FormatGB(usage), utils.FormatGB(limit)))
		info.WriteString(fmt.Sprintf("├ Всего использовано: <%s>\n", utils.FormatGB(user.LifetimeUsedTrafficBytes)))
		untilReset := start.Add(24 * time.Hour).Sub(time.Now())
		info.WriteString(fmt.Sprintf("├ До сброса трафика: <%s>", untilReset.Truncate(time.Second)))
	} else {
		info.WriteString(fmt.Sprintf(h.translation.GetText(lang, "balance_info"), int(customer.Balance)))
	}

	keyboard := [][]models.InlineKeyboardButton{
		{{Text: h.translation.GetText(lang, "topup_button"), CallbackData: CallbackTopup}},
		{{Text: h.translation.GetText(lang, "buy_sub_balance_button"), CallbackData: CallbackBuy}},
		{{Text: h.translation.GetText(lang, "back_button"), CallbackData: CallbackStart}},
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      callback.Chat.ID,
		MessageID:   callback.ID,
		ParseMode:   models.ParseModeHTML,
		Text:        info.String(),
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		slog.Error("Error sending balance message", err)
	}
}

func (h Handler) TopupCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	lang := update.CallbackQuery.From.LanguageCode
	keyboard := [][]models.InlineKeyboardButton{
		{{Text: "100", CallbackData: fmt.Sprintf("%s?amount=100", CallbackTopupMethod)}},
		{{Text: "300", CallbackData: fmt.Sprintf("%s?amount=300", CallbackTopupMethod)}},
		{{Text: "500", CallbackData: fmt.Sprintf("%s?amount=500", CallbackTopupMethod)}},
		{{Text: h.translation.GetText(lang, "back_button"), CallbackData: CallbackBalance}},
	}
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      callback.Chat.ID,
		MessageID:   callback.ID,
		Text:        h.translation.GetText(lang, "topup_button"),
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		slog.Error("Error sending topup message", err)
	}
}

func (h Handler) TopupMethodCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	data := parseCallbackData(update.CallbackQuery.Data)
	amount := data["amount"]
	lang := update.CallbackQuery.From.LanguageCode

	var keyboard [][]models.InlineKeyboardButton
	if config.IsCryptoPayEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(lang, "crypto_button"), CallbackData: fmt.Sprintf("%s?month=0&invoiceType=%s&amount=%s", CallbackPayment, database.InvoiceTypeCrypto, amount)}})
	}
	if config.IsYookasaEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(lang, "card_button"), CallbackData: fmt.Sprintf("%s?month=0&invoiceType=%s&amount=%s", CallbackPayment, database.InvoiceTypeYookasa, amount)}})
	}
	if config.IsTelegramStarsEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(lang, "stars_button"), CallbackData: fmt.Sprintf("%s?month=0&invoiceType=%s&amount=%s", CallbackPayment, database.InvoiceTypeTelegram, amount)}})
	}
	keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(lang, "back_button"), CallbackData: CallbackTopup}})

	_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      callback.Chat.ID,
		MessageID:   callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		slog.Error("Error sending topup methods", err)
	}
}

func (h Handler) PayFromBalanceCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	data := parseCallbackData(update.CallbackQuery.Data)
	month, _ := strconv.Atoi(data["month"])

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctxTimeout, callback.Chat.ID)
	if err != nil || customer == nil {
		return
	}
	if err := h.paymentService.PurchaseFromBalance(ctxTimeout, customer, month); err != nil {
		slog.Error("error pay from balance", err)
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
