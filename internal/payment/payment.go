package payment

import (
	"context"
	"fmt"
	remapi "github.com/Jolymmiles/remnawave-api-go/api"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"log/slog"
	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/cryptopay"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/yookasa"
	"remnawave-tg-shop-bot/utils"
	"time"
)

type PaymentService struct {
	purchaseRepository  *database.PurchaseRepository
	remnawaveClient     *remnawave.Client
	customerRepository  *database.CustomerRepository
	telegramBot         *bot.Bot
	translation         *translation.Manager
	cryptoPayClient     *cryptopay.Client
	yookasaClient       *yookasa.Client
	referralRepository  *database.ReferralRepository
	promocodeRepository *database.PromocodeRepository
	cache               *cache.Cache
}

func NewPaymentService(
	translation *translation.Manager,
	purchaseRepository *database.PurchaseRepository,
	remnawaveClient *remnawave.Client,
	customerRepository *database.CustomerRepository,
	telegramBot *bot.Bot,
	cryptoPayClient *cryptopay.Client,
	yookasaClient *yookasa.Client,
	referralRepository *database.ReferralRepository,
	promocodeRepository *database.PromocodeRepository,
	cache *cache.Cache,
) *PaymentService {
	return &PaymentService{
		purchaseRepository:  purchaseRepository,
		remnawaveClient:     remnawaveClient,
		customerRepository:  customerRepository,
		telegramBot:         telegramBot,
		translation:         translation,
		cryptoPayClient:     cryptoPayClient,
		yookasaClient:       yookasaClient,
		referralRepository:  referralRepository,
		promocodeRepository: promocodeRepository,
		cache:               cache,
	}
}

func (s PaymentService) ProcessPurchaseById(ctx context.Context, purchaseId int64) error {
	purchase, err := s.purchaseRepository.FindById(ctx, purchaseId)
	if err != nil {
		return err
	}
	if purchase == nil {
		return fmt.Errorf("purchase with crypto invoice id %d not found", utils.MaskHalfInt64(purchaseId))
	}

	customer, err := s.customerRepository.FindById(ctx, purchase.CustomerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return fmt.Errorf("customer %s not found", utils.MaskHalfInt64(purchase.CustomerID))
	}

	if messageId, b := s.cache.Get(purchase.ID); b {
		_, err = s.telegramBot.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    customer.TelegramID,
			MessageID: messageId,
		})
		if err != nil {
			slog.Error("Error deleting message", err)
		}
	}

	err = s.purchaseRepository.MarkAsPaid(ctx, purchase.ID)
	if err != nil {
		return err
	}

	_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: customer.TelegramID,
		Text:   fmt.Sprintf(s.translation.GetText(customer.Language, "balance_topped_up"), int(purchase.Amount)),
	})
	if err != nil {
		return err
	}

	slog.Info("purchase processed", "purchase_id", utils.MaskHalfInt64(purchase.ID), "type", purchase.InvoiceType, "customer_id", utils.MaskHalfInt64(customer.ID))

	return nil
}

func (s PaymentService) PurchaseFromBalance(ctx context.Context, customer *database.Customer, months int) error {
	price := config.Price(months)
	if customer.Balance < float64(price) {
		_, _ = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{ChatID: customer.TelegramID, Text: s.translation.GetText(customer.Language, "insufficient_balance")})
		return nil
	}

	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, customer.ID, customer.TelegramID, config.TrafficLimit(), months*30)
	if err != nil {
		return err
	}

	newBalance := customer.Balance - float64(price)
	updates := map[string]interface{}{
		"subscription_link": user.SubscriptionUrl,
		"expire_at":         user.ExpireAt,
		"balance":           newBalance,
	}

	customer.SubscriptionLink = &user.SubscriptionUrl
	customer.ExpireAt = &user.ExpireAt
	customer.Balance = newBalance

	if err := s.customerRepository.UpdateFields(ctx, customer.ID, updates); err != nil {
		return err
	}

	_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      customer.TelegramID,
		ParseMode:   models.ParseModeHTML,
		Text:        s.translation.GetText(customer.Language, "subscription_activated"),
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: s.createConnectKeyboard(customer)},
	})
	return err
}

func (s PaymentService) createConnectKeyboard(customer *database.Customer) [][]models.InlineKeyboardButton {
	var inlineCustomerKeyboard [][]models.InlineKeyboardButton

	if config.GetMiniAppURL() != "" {
		inlineCustomerKeyboard = append(inlineCustomerKeyboard, []models.InlineKeyboardButton{
			{Text: s.translation.GetText(customer.Language, "connect_button"), WebApp: &models.WebAppInfo{
				URL: config.GetMiniAppURL(),
			}},
		})
	} else {
		inlineCustomerKeyboard = append(inlineCustomerKeyboard, []models.InlineKeyboardButton{
			{Text: s.translation.GetText(customer.Language, "connect_button"), CallbackData: "connect"},
		})
	}

	inlineCustomerKeyboard = append(inlineCustomerKeyboard, []models.InlineKeyboardButton{
		{Text: s.translation.GetText(customer.Language, "back_button"), CallbackData: "start"},
	})
	return inlineCustomerKeyboard
}

func (s PaymentService) CreatePurchase(ctx context.Context, amount int, months int, customer *database.Customer, invoiceType database.InvoiceType) (url string, purchaseId int64, err error) {
	switch invoiceType {
	case database.InvoiceTypeCrypto:
		return s.createCryptoInvoice(ctx, amount, months, customer)
	case database.InvoiceTypeYookasa:
		return s.createYookasaInvoice(ctx, amount, months, customer)
	case database.InvoiceTypeTelegram:
		return s.createTelegramInvoice(ctx, amount, months, customer)
	case database.InvoiceTypeTribute:
		return s.createTributeInvoice(ctx, amount, months, customer)
	default:
		return "", 0, fmt.Errorf("unknown invoice type: %s", invoiceType)
	}
}

func (s PaymentService) createCryptoInvoice(ctx context.Context, amount int, months int, customer *database.Customer) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeCrypto,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(amount),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return "", 0, err
	}

	invoice, err := s.cryptoPayClient.CreateInvoice(&cryptopay.InvoiceRequest{
		CurrencyType:   "fiat",
		Fiat:           "RUB",
		Amount:         fmt.Sprintf("%d", amount),
		AcceptedAssets: "USDT",
		Payload:        fmt.Sprintf("purchaseId=%d&username=%s", purchaseId, ctx.Value("username")),
		Description:    fmt.Sprintf("Subscription on %d month", months),
		PaidBtnName:    "callback",
		PaidBtnUrl:     config.BotURL(),
	})
	if err != nil {
		slog.Error("Error creating invoice", err)
		return "", 0, err
	}

	updates := map[string]interface{}{
		"crypto_invoice_url": invoice.BotInvoiceUrl,
		"crypto_invoice_id":  invoice.InvoiceID,
		"status":             database.PurchaseStatusPending,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return "", 0, err
	}

	return invoice.BotInvoiceUrl, purchaseId, nil
}

func (s PaymentService) createYookasaInvoice(ctx context.Context, amount int, months int, customer *database.Customer) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeYookasa,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(amount),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return "", 0, err
	}

	invoice, err := s.yookasaClient.CreateInvoice(ctx, amount, months, customer.ID, purchaseId)
	if err != nil {
		slog.Error("Error creating invoice", err)
		return "", 0, err
	}

	updates := map[string]interface{}{
		"yookasa_url": invoice.Confirmation.ConfirmationURL,
		"yookasa_id":  invoice.ID,
		"status":      database.PurchaseStatusPending,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return "", 0, err
	}

	return invoice.Confirmation.ConfirmationURL, purchaseId, nil
}

func (s PaymentService) createTelegramInvoice(ctx context.Context, amount int, months int, customer *database.Customer) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeTelegram,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(amount),
		Currency:    "STARS",
		CustomerID:  customer.ID,
		Month:       months,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return "", 0, nil
	}

	invoiceUrl, err := s.telegramBot.CreateInvoiceLink(ctx, &bot.CreateInvoiceLinkParams{
		Title:    s.translation.GetText(customer.Language, "invoice_title"),
		Currency: "XTR",
		Prices: []models.LabeledPrice{
			{
				Label:  s.translation.GetText(customer.Language, "invoice_label"),
				Amount: amount,
			},
		},
		Description: s.translation.GetText(customer.Language, "invoice_description"),
		Payload:     fmt.Sprintf("%d&%s", purchaseId, ctx.Value("username")),
	})

	updates := map[string]interface{}{
		"status": database.PurchaseStatusPending,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", err)
		return "", 0, err
	}

	return invoiceUrl, purchaseId, nil
}

func (s PaymentService) ActivateTrial(ctx context.Context, telegramId int64) (string, error) {
	if config.TrialDays() == 0 {
		return "", nil
	}
	customer, err := s.customerRepository.FindByTelegramId(ctx, telegramId)
	if err != nil {
		slog.Error("Error finding customer", err)
		return "", err
	}
	if customer == nil {
		return "", fmt.Errorf("customer %d not found", telegramId)
	}
	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, customer.ID, telegramId, config.TrialTrafficLimit(), config.TrialDays())
	if err != nil {
		slog.Error("Error creating user", err)
		return "", err
	}

	customerFilesToUpdate := map[string]interface{}{
		"subscription_link": user.GetSubscriptionUrl(),
		"expire_at":         user.GetExpireAt(),
	}

	err = s.customerRepository.UpdateFields(ctx, customer.ID, customerFilesToUpdate)
	if err != nil {
		return "", err
	}

	return user.GetSubscriptionUrl(), nil

}

func (s PaymentService) CancelPayment(purchaseId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	purchase, err := s.purchaseRepository.FindById(ctx, purchaseId)
	if err != nil {
		return err
	}
	if purchase == nil {
		return fmt.Errorf("purchase with crypto invoice id %d not found", utils.MaskHalfInt64(purchaseId))
	}

	purchaseFieldsToUpdate := map[string]interface{}{
		"status": database.PurchaseStatusCancel,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, purchaseFieldsToUpdate)
	if err != nil {
		return err
	}

	return nil
}

func (s PaymentService) createTributeInvoice(ctx context.Context, amount int, months int, customer *database.Customer) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeTribute,
		Status:      database.PurchaseStatusPending,
		Amount:      float64(amount),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
	})
	if err != nil {
		slog.Error("Error creating purchase", err)
		return "", 0, err
	}

	return "", purchaseId, nil
}

func (s PaymentService) GetUser(ctx context.Context, telegramId int64) (*remapi.UserDto, error) {
	return s.remnawaveClient.GetUserByTelegramID(ctx, telegramId)
}

func (s PaymentService) GetUserDailyUsage(ctx context.Context, uuid string, start, end time.Time) (float64, error) {
	return s.remnawaveClient.GetUserDailyUsage(ctx, uuid, start, end)
}
func (s PaymentService) CreatePromocode(ctx context.Context, customer *database.Customer, months, uses int) (string, error) {
	cost := config.Price(months) * uses
	if !config.IsAdmin(customer.TelegramID) {
		if customer.Balance < float64(cost) {
			return "", fmt.Errorf("insufficient balance")
		}
		newBalance := customer.Balance - float64(cost)
		if err := s.customerRepository.UpdateFields(ctx, customer.ID, map[string]interface{}{"balance": newBalance}); err != nil {
			return "", err
		}
		customer.Balance = newBalance
	}

	code := uuid.New().String()[:8]
	_, err := s.promocodeRepository.Create(ctx, &database.Promocode{
		Code:      code,
		Months:    months,
		UsesLeft:  uses,
		CreatedBy: customer.TelegramID,
	})
	if err != nil {
		return "", err
	}
	return code, nil
}
