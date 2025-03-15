package handler

import (
	"context"
	"fmt"
	"github.com/biter777/countries"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/cryptopay"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/yookasa"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	customerRepository *database.CustomerRepository
	purchaseRepository *database.PurchaseRepository
	cryptoPayClient    *cryptopay.Client
	yookasaClient      *yookasa.Client
	translation        *translation.Manager
	paymentService     *payment.PaymentService
}

func NewHandler(
	paymentService *payment.PaymentService,
	translation *translation.Manager,
	customerRepository *database.CustomerRepository,
	purchaseRepository *database.PurchaseRepository,
	cryptoPayClient *cryptopay.Client,
	yookasaClient *yookasa.Client) *Handler {
	return &Handler{
		paymentService:     paymentService,
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		cryptoPayClient:    cryptoPayClient,
		yookasaClient:      yookasaClient,
		translation:        translation,
	}
}

const (
	CallbackBuy           = "buy"
	CallbackSell          = "sell"
	CallbackStart         = "start"
	CallbackCrypto        = "crypto"
	CallbackCard          = "card"
	CallbackConnect       = "connect"
	CallbackTelegramStars = "telegram_stars"
	CallbackTrial         = "trial"
)
func (h Handler) StartCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctxWithTime, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	langCode := update.Message.From.LanguageCode
	existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("error finding customer by telegram id", err)
	}

	if existingCustomer == nil {
		err := h.customerRepository.Create(ctxWithTime, &database.Customer{
			TelegramID: update.Message.Chat.ID,
			Language:   langCode,
		})
		if err != nil {
			slog.Error("error creating customer", err)
			return
		}
		existingCustomer, err = h.customerRepository.FindByTelegramId(ctx, update.Message.Chat.ID)
		if err != nil {
			slog.Error("error fetching newly created customer", err)
			return
		}
		slog.Info("user created", "telegramId", update.Message.Chat.ID)
	} else {
		updates := map[string]interface{}{
			"language": langCode,
		}

		err = h.customerRepository.UpdateFields(ctx, existingCustomer.ID, updates)
		if err != nil {
			slog.Error("Error updating customer", err)
			return
		}
	}

	inlineKeyboard := [][]models.InlineKeyboardButton{
		{{Text: h.translation.GetText(langCode, "buy_button"), CallbackData: "buy"}},
		{{Text: h.translation.GetText(langCode, "connect_button"), CallbackData: "connect"}},
	}

	if config.ServerStatusURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "server_status_button"), URL: config.ServerStatusURL()},
		})
	}

	if config.SupportURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "support_button"), URL: config.SupportURL()},
		})
	}

	if config.FeedbackURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "feedback_button"), URL: config.FeedbackURL()},
		})
	}

	if config.ChannelURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "channel_button"), URL: config.ChannelURL()},
		})
	}
	if config.TrialTrafficLimitGB() > 0  && existingCustomer.SubscriptionLink == nil {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "trial_button"), CallbackData: "trial"},
		})
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: fmt.Sprintf(h.translation.GetText(langCode, "greeting"), bot.EscapeMarkdown(buildAvailableCountriesLists(langCode))),
	})
	if err != nil {
		slog.Error("Error sending /start message", err)
	}
}

func buildAvailableCountriesLists(langCode string) string {
	var locationsText strings.Builder
	countriesMap := config.Countries()

	keys := make([]string, 0, len(countriesMap))
	for k := range countriesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		country := strings.Split(countriesMap[k], " ")
		if i == len(keys)-1 {
			if langCode == "ru" {
				locationsText.WriteString(fmt.Sprintf("└ %s%s\n", country[0], countries.ByName(country[1]).StringRus()))
			} else {
				locationsText.WriteString(fmt.Sprintf("└ %s%s\n", country[0], countries.ByName(country[1]).String()))
			}
		} else {
			if langCode == "ru" {
				locationsText.WriteString(fmt.Sprintf("├ %s%s\n", country[0], countries.ByName(country[1]).StringRus()))
			} else {
				locationsText.WriteString(fmt.Sprintf("├ %s%s\n", country[0], countries.ByName(country[1]).String()))
			}
		}
	}

	return locationsText.String()
}

func (h Handler) StartCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode
	inlineKeyboard := [][]models.InlineKeyboardButton{
		{{Text: h.translation.GetText(langCode, "buy_button"), CallbackData: "buy"}},
		{{Text: h.translation.GetText(langCode, "connect_button"), CallbackData: "connect"}},
	}
	existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer by telegram id", err)
	}
	if config.TrialTrafficLimitGB() > 0 &&  existingCustomer.SubscriptionLink == nil {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "trial_button"), CallbackData: "trial"},
		})
	}


	if config.ServerStatusURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "server_status_button"), URL: config.ServerStatusURL()},
		})
	}

	if config.SupportURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "support_button"), URL: config.SupportURL()},
		})
	}

	if config.FeedbackURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "feedback_button"), URL: config.FeedbackURL()},
		})
	}

	if config.ChannelURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "channel_button"), URL: config.ChannelURL()},
		})
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: fmt.Sprintf(h.translation.GetText(langCode, "greeting"), bot.EscapeMarkdown(buildAvailableCountriesLists(langCode))),
	})
	if err != nil {
		slog.Error("Error sending /start message", err)
	}
}

func (h Handler) BuyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	langCode := update.CallbackQuery.From.LanguageCode

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(langCode, "month_1"), CallbackData: buildSellCallbackData(1)},
					{Text: h.translation.GetText(langCode, "month_3"), CallbackData: buildSellCallbackData(3)},
					{Text: h.translation.GetText(langCode, "month_6"), CallbackData: buildSellCallbackData(6)},
				},
				{
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
				},
			},
		},
		Text: fmt.Sprintf(h.translation.GetText(langCode, "pricing_info"),
			calculatePrice(1),
			calculatePrice(2),
			calculatePrice(3)),
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

	var keyboard [][]models.InlineKeyboardButton

	if config.IsCryptoPayEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "crypto_button"), CallbackData: fmt.Sprintf("%s?month=%s", CallbackCrypto, month)},
		})
	}

	if config.IsYookasaEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "card_button"), CallbackData: fmt.Sprintf("%s?month=%s", CallbackCard, month)},
		})
	}

	if config.IsTelegramStarsEnabled() {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: "⭐Telegram Stars", CallbackData: fmt.Sprintf("%s?month=%s", CallbackTelegramStars, month)},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
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

func (h Handler) CryptoCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	month, err := strconv.Atoi(callbackQuery["month"])
	if err != nil {
		slog.Error("Error getting month from query", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", err)
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	price := calculatePrice(month)

	purchaseId, err := h.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeCrypto,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(price),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       month,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return
	}

	invoice, err := h.cryptoPayClient.CreateInvoice(&cryptopay.InvoiceRequest{
		CurrencyType:   "fiat",
		Fiat:           "RUB",
		Amount:         fmt.Sprintf("%d", price),
		AcceptedAssets: "USDT",
		Payload:        fmt.Sprintf("customerId=%d&purchaseId=%d", callback.Chat.ID, purchaseId),
		Description:    fmt.Sprintf("Subscription on %d month", month),
		PaidBtnName:    "callback",
		PaidBtnUrl:     config.BotURL(),
	})
	if err != nil {
		slog.Error("Error creating invoice", err)
		return
	}

	updates := map[string]interface{}{
		"crypto_invoice_url": invoice.BotInvoiceUrl,
		"crypto_invoice_id":  invoice.InvoiceID,
		"status":             database.PurchaseStatusPending,
	}

	err = h.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return
	}

	langCode := update.CallbackQuery.From.LanguageCode

	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(langCode, "pay_button"), URL: invoice.BotInvoiceUrl},
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: buildSellCallbackData(month)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error updating sell message", err)
	}

}


func (h Handler) TrialCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer by telegram id", err)
	}
	
	err = h.paymentService.ActivateTrialSubscription(existingCustomer.TelegramID)
	if err != nil {
		slog.Error("Error activating trial", err)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: existingCustomer.TelegramID,
			Text:   h.translation.GetText(existingCustomer.Language, "trial_subscription_error"),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: h.translation.GetText(existingCustomer.Language, "back_button"), CallbackData: CallbackStart}},
				},
			},
		})
		if err != nil {
			slog.Error("Error sending trial subscription error message", err)
		}
		return

	}
	
	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error creating or updating user", err)
		return
	}
	
	subscriptionLink := ""
	if customer.SubscriptionLink != nil {
		subscriptionLink = *customer.SubscriptionLink
	}
	
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: existingCustomer.TelegramID,
		Text:   h.translation.GetText(existingCustomer.Language, "trial_subscription_activated"),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(existingCustomer.Language, "connect_button"), URL: subscriptionLink},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error sending trial subscription activated message", err)
	}
}

func (h Handler) YookasaCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	month, err := strconv.Atoi(callbackQuery["month"])
	if err != nil {
		slog.Error("Error getting month from query", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", err)
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	price := calculatePrice(month)
	purchaseId, err := h.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeYookasa,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(price),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       month,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return
	}
	langCode := update.CallbackQuery.From.LanguageCode

	invoice, err := h.yookasaClient.CreateInvoice(ctx, price, month, customer.ID, purchaseId)
	if err != nil {
		slog.Error("Error creating invoice", err)
		return
	}

	updates := map[string]interface{}{
		"yookasa_url": invoice.Confirmation.ConfirmationURL,
		"yookasa_id":  invoice.ID,
		"status":      database.PurchaseStatusPending,
	}

	err = h.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return
	}

	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(langCode, "pay_button"), URL: invoice.Confirmation.ConfirmationURL},
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: buildSellCallbackData(month)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error updating sell message", err)
	}
}

func (h Handler) ConnectCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	customer, err := h.customerRepository.FindByTelegramId(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", err)
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", update.Message.Chat.ID, "error", err)
	}

	langCode := update.Message.From.LanguageCode

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   buildConnectText(customer, langCode),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
			},
		},
	})

	if err != nil {
		slog.Error("Error sending connect message", err)
	}
}

func (h Handler) ConnectCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message

	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", err)
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
	}

	langCode := update.CallbackQuery.From.LanguageCode

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		Text:      buildConnectText(customer, langCode),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
			},
		},
	})

	if err != nil {
		slog.Error("Error sending connect message", err)
	}
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
	purchaseId, err := strconv.Atoi(update.Message.SuccessfulPayment.InvoicePayload)
	if err != nil {
		slog.Error("Error parsing purchase id", err)
	}

	err = h.paymentService.ProcessPurchaseById(int64(purchaseId))
	if err != nil {
		slog.Error("Error processing purchase", err)
	}

}

func (h Handler) TelegramStarsCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery.Message.Message
	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	month, err := strconv.Atoi(callbackQuery["month"])
	if err != nil {
		slog.Error("Error getting month from query", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", err)
	}
	if customer == nil {
		slog.Error("customer not exist", "chatID", callback.Chat.ID, "error", err)
		return
	}

	price := calculatePrice(month)
	purchaseId, err := h.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeTelegram,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(price),
		Currency:    "STARS",
		CustomerID:  customer.ID,
		Month:       month,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return
	}
	langCode := update.CallbackQuery.From.LanguageCode

	invoiceUrl, err := b.CreateInvoiceLink(ctx, &bot.CreateInvoiceLinkParams{
		Title:    h.translation.GetText(langCode, "invoice_title"),
		Currency: "XTR",
		Prices: []models.LabeledPrice{
			{
				Label:  h.translation.GetText(langCode, "invoice_label"),
				Amount: price,
			},
		},
		Description: h.translation.GetText(langCode, "invoice_description"),
		Payload:     strconv.FormatInt(purchaseId, 10),
	})

	updates := map[string]interface{}{
		"status": database.PurchaseStatusPending,
	}

	err = h.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return
	}

	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: h.translation.GetText(langCode, "pay_button"), URL: invoiceUrl},
					{Text: h.translation.GetText(langCode, "back_button"), CallbackData: buildSellCallbackData(month)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Error updating sell message", err)
	}
}

func buildConnectText(customer *database.Customer, langCode string) string {
	var info strings.Builder

	tm := translation.GetInstance()

	if customer.ExpireAt != nil {
		currentTime := time.Now()

		if currentTime.Before(*customer.ExpireAt) {
			formattedDate := customer.ExpireAt.Format("02.01.2006 15:04")

			subscriptionActiveText := tm.GetText(langCode, "subscription_active")
			info.WriteString(fmt.Sprintf(subscriptionActiveText, formattedDate))

			if customer.SubscriptionLink != nil && *customer.SubscriptionLink != "" {
				subscriptionLinkText := tm.GetText(langCode, "subscription_link")
				info.WriteString(fmt.Sprintf(subscriptionLinkText, *customer.SubscriptionLink))
			}
		} else {
			noSubscriptionText := tm.GetText(langCode, "no_subscription")
			info.WriteString(noSubscriptionText)
		}
	} else {
		noSubscriptionText := tm.GetText(langCode, "no_subscription")
		info.WriteString(noSubscriptionText)
	}

	return info.String()
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

func buildSellCallbackData(month int) string {
	return fmt.Sprintf("%s?month=%d", CallbackSell, month)
}

func calculatePrice(month int) int {
	return config.Price() * month
}
