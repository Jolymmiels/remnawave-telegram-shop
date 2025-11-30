package payment

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

type PaymentService struct {
	purchaseRepository *database.PurchaseRepository
	remnawaveClient    *remnawave.Client
	customerRepository *database.CustomerRepository
	telegramBot        *bot.Bot
	translation        *translation.Manager
	cryptoPayClient    *cryptopay.Client
	yookasaClient      *yookasa.Client
	referralRepository *database.ReferralRepository
	cache              *cache.Cache
	planRepository     *database.PlanRepository
	settingsRepository *database.SettingsRepository
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
	cache *cache.Cache,
	planRepository *database.PlanRepository,
	settingsRepository *database.SettingsRepository,
) *PaymentService {
	return &PaymentService{
		purchaseRepository: purchaseRepository,
		remnawaveClient:    remnawaveClient,
		customerRepository: customerRepository,
		telegramBot:        telegramBot,
		translation:        translation,
		cryptoPayClient:    cryptoPayClient,
		yookasaClient:      yookasaClient,
		referralRepository: referralRepository,
		cache:              cache,
		planRepository:     planRepository,
		settingsRepository: settingsRepository,
	}
}

func (s PaymentService) ProcessPurchaseById(ctx context.Context, purchaseId int64) error {
	// Atomically lock the purchase for processing to prevent race conditions
	purchase, err := s.purchaseRepository.LockForProcessing(ctx, purchaseId)
	if err != nil {
		return fmt.Errorf("failed to lock purchase: %w", err)
	}
	if purchase == nil {
		// Already being processed or already processed by another worker
		slog.Debug("Purchase already locked or processed", "purchaseId", utils.MaskHalfInt64(purchaseId))
		return nil
	}

	// If processing fails, unlock the purchase so it can be retried
	defer func() {
		if err != nil {
			if unlockErr := s.purchaseRepository.UnlockPurchase(ctx, purchaseId); unlockErr != nil {
				slog.Error("Failed to unlock purchase after error", "purchaseId", purchaseId, "error", unlockErr)
			}
		}
	}()

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
			slog.Error("Error deleting message", "error", err)
		}
	}

	// Get plan settings or use defaults
	var plan *database.Plan
	if purchase.PlanID != nil {
		plan, _ = s.planRepository.FindById(ctx, *purchase.PlanID)
	}
	if plan == nil {
		plan, _ = s.planRepository.FindDefault(ctx)
	}

	trafficLimit := config.TrafficLimit()
	if plan != nil && plan.TrafficLimit > 0 {
		trafficLimit = plan.TrafficLimit * 1024 * 1024 * 1024 // Convert GB to bytes
	}

	user, err := s.remnawaveClient.CreateOrUpdateUserWithPlan(ctx, customer.ID, customer.TelegramID, trafficLimit, purchase.Month*config.DaysInMonth(), false, plan)
	if err != nil {
		return err
	}

	err = s.purchaseRepository.MarkAsPaid(ctx, purchase.ID)
	if err != nil {
		return err
	}

	customerFilesToUpdate := map[string]interface{}{
		"subscription_link": user.SubscriptionUrl,
		"expire_at":         user.ExpireAt,
	}

	err = s.customerRepository.UpdateFields(ctx, customer.ID, customerFilesToUpdate)
	if err != nil {
		return err
	}

	_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: customer.TelegramID,
		Text:   s.translation.GetText(customer.Language, "subscription_activated"),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: s.createConnectKeyboard(customer),
		},
	})
	if err != nil {
		return err
	}

	// Process referral bonus (uses settings from DB)
	if err := s.processReferralBonus(ctx, customer, purchase.ID, purchase.Month); err != nil {
		slog.Error("Error processing referral bonus", "error", err)
	}
	slog.Info("purchase processed", "purchase_id", utils.MaskHalfInt64(purchase.ID), "type", purchase.InvoiceType, "customer_id", utils.MaskHalfInt64(customer.ID))

	return nil
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

func (s PaymentService) CreatePurchase(ctx context.Context, amount float64, months int, customer *database.Customer, invoiceType database.InvoiceType, planID *int64) (url string, purchaseId int64, err error) {
	switch invoiceType {
	case database.InvoiceTypeCrypto:
		return s.createCryptoInvoice(ctx, amount, months, customer, planID)
	case database.InvoiceTypeYookassa:
		return s.createYookasaInvoice(ctx, amount, months, customer, planID)
	case database.InvoiceTypeTelegram:
		return s.createTelegramInvoice(ctx, amount, months, customer, planID)
	case database.InvoiceTypeTribute:
		return s.createTributeInvoice(ctx, amount, months, customer, planID)
	default:
		return "", 0, fmt.Errorf("unknown invoice type: %s", invoiceType)
	}
}

var ErrCustomerNotFound = errors.New("customer not found")

func (s PaymentService) CancelTributePurchase(ctx context.Context, telegramId int64) error {
	slog.Info("Canceling tribute purchase", "telegram_id", utils.MaskHalfInt64(telegramId))
	customer, err := s.customerRepository.FindByTelegramId(ctx, telegramId)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	tributePurchase, err := s.purchaseRepository.FindByCustomerIDAndInvoiceTypeLast(ctx, customer.ID, database.InvoiceTypeTribute)
	if err != nil {
		return err
	}
	if tributePurchase == nil {
		return errors.New("tribute purchase not found")
	}
	expireAt, err := s.remnawaveClient.DecreaseSubscription(ctx, telegramId, config.TrafficLimit(), -tributePurchase.Month*config.DaysInMonth())
	if err != nil {
		return err
	}

	if err := s.customerRepository.UpdateFields(ctx, customer.ID, map[string]interface{}{
		"expire_at": expireAt,
	}); err != nil {
		return err
	}

	if err := s.purchaseRepository.UpdateFields(ctx, tributePurchase.ID, map[string]interface{}{
		"status": database.PurchaseStatusCancel,
	}); err != nil {
		return err
	}
	_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    telegramId,
		ParseMode: models.ParseModeHTML,
		Text:      s.translation.GetText(customer.Language, "tribute_cancelled"),
	})
	if err != nil {
		slog.Error("Error sending message about tribute cancelled", "error", err, "telegram_id", utils.MaskHalfInt64(telegramId))
	}
	slog.Info("Canceled tribute purchase", "purchase_id", utils.MaskHalfInt64(tributePurchase.ID), "telegram_id", utils.MaskHalfInt64(telegramId))
	return nil
}

// getPaymentReturnURL returns the payment return URL from settings or falls back to config.BotURL()
func (s PaymentService) getPaymentReturnURL() string {
	if s.settingsRepository != nil {
		if url := s.settingsRepository.Get("payment_return_url"); url != "" {
			return url
		}
	}
	return config.BotURL()
}

func (s PaymentService) createCryptoInvoice(ctx context.Context, amount float64, months int, customer *database.Customer, planID *int64) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeCrypto,
		Status:      database.PurchaseStatusNew,
		Amount:      amount,
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
		PlanID:      planID,
	})
	if err != nil {
		slog.Error("Error creating purchase", "error", err)
		return "", 0, err
	}

	invoice, err := s.cryptoPayClient.CreateInvoice(&cryptopay.InvoiceRequest{
		CurrencyType:   "fiat",
		Fiat:           "RUB",
		Amount:         fmt.Sprintf("%d", int(amount)),
		AcceptedAssets: "USDT",
		Payload:        fmt.Sprintf("purchaseId=%d&username=%s", purchaseId, ctx.Value("username")),
		Description:    fmt.Sprintf("Subscription on %d month", months),
		PaidBtnName:    "callback",
		PaidBtnUrl:     s.getPaymentReturnURL(),
	})
	if err != nil {
		slog.Error("Error creating invoice", "error", err)
		return "", 0, err
	}

	updates := map[string]interface{}{
		"crypto_invoice_url": invoice.BotInvoiceUrl,
		"crypto_invoice_id":  invoice.InvoiceID,
		"status":             database.PurchaseStatusPending,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", "error", err)
		return "", 0, err
	}

	return invoice.BotInvoiceUrl, purchaseId, nil
}

func (s PaymentService) createYookasaInvoice(ctx context.Context, amount float64, months int, customer *database.Customer, planID *int64) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeYookassa,
		Status:      database.PurchaseStatusNew,
		Amount:      amount,
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
		PlanID:      planID,
	})
	if err != nil {
		slog.Error("Error creating purchase", "error", err)
		return "", 0, err
	}

	// Check if autopay is enabled to save payment method for future recurring payments
	var invoice *yookasa.Payment
	if s.settingsRepository.GetBool("recurring_payments_enabled", false) {
		invoice, err = s.yookasaClient.CreateInvoiceWithSavePaymentMethod(ctx, int(amount), months, customer.ID, purchaseId, s.getPaymentReturnURL())
	} else {
		invoice, err = s.yookasaClient.CreateInvoice(ctx, int(amount), months, customer.ID, purchaseId, s.getPaymentReturnURL())
	}
	if err != nil {
		slog.Error("Error creating invoice", "error", err)
		return "", 0, err
	}

	updates := map[string]interface{}{
		"yookasa_url": invoice.Confirmation.ConfirmationURL,
		"yookasa_id":  invoice.ID,
		"status":      database.PurchaseStatusPending,
	}

	err = s.purchaseRepository.UpdateFields(ctx, purchaseId, updates)
	if err != nil {
		slog.Error("Error updating purchase", "error", err)
		return "", 0, err
	}

	return invoice.Confirmation.ConfirmationURL, purchaseId, nil
}

func (s PaymentService) createTelegramInvoice(ctx context.Context, amount float64, months int, customer *database.Customer, planID *int64) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeTelegram,
		Status:      database.PurchaseStatusNew,
		Amount:      amount,
		Currency:    "STARS",
		CustomerID:  customer.ID,
		Month:       months,
		PlanID:      planID,
	})
	if err != nil {
		slog.Error("Error creating purchase", "error", err)
		return "", 0, nil
	}

	invoiceUrl, err := s.telegramBot.CreateInvoiceLink(ctx, &bot.CreateInvoiceLinkParams{
		Title:    s.translation.GetText(customer.Language, "invoice_title"),
		Currency: "XTR",
		Prices: []models.LabeledPrice{
			{
				Label:  s.translation.GetText(customer.Language, "invoice_label"),
				Amount: int(amount),
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
		slog.Error("Error updating purchase", "error", err)
		return "", 0, err
	}

	return invoiceUrl, purchaseId, nil
}

func (s PaymentService) ActivateTrial(ctx context.Context, telegramId int64) (string, error) {
	if !config.IsTrialEnabled() {
		return "", nil
	}
	customer, err := s.customerRepository.FindByTelegramId(ctx, telegramId)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return "", err
	}
	if customer == nil {
		return "", fmt.Errorf("customer %d not found", telegramId)
	}
	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, customer.ID, telegramId, config.TrialTrafficLimit(), config.TrialDays(), true)
	if err != nil {
		slog.Error("Error creating user", "error", err)
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

func (s PaymentService) CancelYookassaPayment(purchaseId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	purchase, err := s.purchaseRepository.FindById(ctx, purchaseId)
	if err != nil {
		return err
	}
	if purchase == nil {
		return fmt.Errorf("purchase with crypto invoice id %s not found", utils.MaskHalfInt64(purchaseId))
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

func (s PaymentService) createTributeInvoice(ctx context.Context, amount float64, months int, customer *database.Customer, planID *int64) (url string, purchaseId int64, err error) {
	purchaseId, err = s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeTribute,
		Status:      database.PurchaseStatusPending,
		Amount:      amount,
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
		PlanID:      planID,
	})
	if err != nil {
		slog.Error("Error creating purchase", "error", err)
		return "", 0, err
	}

	return "", purchaseId, nil
}

func (s *PaymentService) ExtendSubscription(ctx context.Context, customerID int64, bonusDays int) error {
	customer, err := s.customerRepository.FindById(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to find customer: %w", err)
	}
	if customer == nil {
		return fmt.Errorf("customer not found")
	}

	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, customer.ID, customer.TelegramID, config.TrafficLimit(), bonusDays, false)
	if err != nil {
		slog.Error("Error creating user", "error", err)
		return err
	}

	customerFilesToUpdate := map[string]interface{}{
		"subscription_link": user.GetSubscriptionUrl(),
		"expire_at":         user.GetExpireAt(),
	}

	err = s.customerRepository.UpdateFields(ctx, customer.ID, customerFilesToUpdate)
	if err != nil {
		return err
	}

	return nil
}

// getReferralBonusDays calculates the referral bonus days based on settings and tiers
func (s *PaymentService) getReferralBonusDays(ctx context.Context, referrerTelegramID int64) int {
	if !s.settingsRepository.GetBool("referral_enabled", true) {
		return 0
	}

	baseBonusDays := s.settingsRepository.GetInt("referral_bonus_days", config.GetReferralDays())

	if !s.settingsRepository.GetBool("referral_tiers_enabled", false) {
		return baseBonusDays
	}

	referralCount, err := s.referralRepository.CountByReferrer(ctx, referrerTelegramID)
	if err != nil {
		slog.Error("Error counting referrals for tier", "error", err)
		return baseBonusDays
	}

	tier3Threshold := s.settingsRepository.GetInt("referral_tier3_threshold", 30)
	tier2Threshold := s.settingsRepository.GetInt("referral_tier2_threshold", 15)
	tier1Threshold := s.settingsRepository.GetInt("referral_tier1_threshold", 5)

	if referralCount >= tier3Threshold {
		return s.settingsRepository.GetInt("referral_tier3_bonus", 7)
	}
	if referralCount >= tier2Threshold {
		return s.settingsRepository.GetInt("referral_tier2_bonus", 5)
	}
	if referralCount >= tier1Threshold {
		return s.settingsRepository.GetInt("referral_tier1_bonus", 3)
	}

	return baseBonusDays
}

// getRefereeBonusDays returns the bonus days for the invited user (referee)
func (s *PaymentService) getRefereeBonusDays() int {
	if !s.settingsRepository.GetBool("referral_enabled", true) {
		return 0
	}
	return s.settingsRepository.GetInt("referral_referee_bonus_days", 0)
}

// isRecurringReferralEnabled checks if recurring referral bonuses are enabled
func (s *PaymentService) isRecurringReferralEnabled() bool {
	return s.settingsRepository.GetBool("referral_recurring_enabled", false)
}

// getRecurringReferralPercent returns the percentage of subscription days to grant as recurring bonus
func (s *PaymentService) getRecurringReferralPercent() int {
	return s.settingsRepository.GetInt("referral_recurring_percent", 10)
}

// processReferralBonus handles the referral bonus logic for a purchase
// CreateRecurringPayment creates a recurring payment for a customer with a saved payment method
func (s *PaymentService) CreateRecurringPayment(ctx context.Context, customer *database.Customer) error {
	if customer.PaymentMethodID == nil || customer.AutopayPlanID == nil {
		return fmt.Errorf("customer %d has no payment method or autopay plan", customer.ID)
	}

	plan, err := s.planRepository.FindById(ctx, *customer.AutopayPlanID)
	if err != nil {
		return fmt.Errorf("failed to find plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("plan %d not found", *customer.AutopayPlanID)
	}

	months := customer.AutopayMonths
	if months == 0 {
		months = 1
	}
	price := plan.GetPrice(months)

	// Create purchase record
	purchaseId, err := s.purchaseRepository.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypeYookassa,
		Status:      database.PurchaseStatusNew,
		Amount:      float64(price),
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       months,
		PlanID:      customer.AutopayPlanID,
	})
	if err != nil {
		return fmt.Errorf("failed to create purchase: %w", err)
	}

	// Create recurring payment via YooKassa
	paymentMethodID, err := uuid.Parse(*customer.PaymentMethodID)
	if err != nil {
		return fmt.Errorf("invalid payment method ID: %w", err)
	}

	payment, err := s.yookasaClient.CreateRecurringPayment(ctx, price, months, customer.ID, purchaseId, paymentMethodID)
	if err != nil {
		// Mark purchase as failed
		_ = s.purchaseRepository.UpdateFields(ctx, purchaseId, map[string]interface{}{
			"status": database.PurchaseStatusCancel,
		})
		return fmt.Errorf("failed to create recurring payment: %w", err)
	}

	// Update purchase with YooKassa data
	updates := map[string]interface{}{
		"yookasa_id": payment.ID,
		"status":     database.PurchaseStatusPending,
	}
	if err := s.purchaseRepository.UpdateFields(ctx, purchaseId, updates); err != nil {
		return fmt.Errorf("failed to update purchase: %w", err)
	}

	slog.Info("Created recurring payment",
		"customer_id", customer.ID,
		"purchase_id", purchaseId,
		"payment_id", utils.MaskHalf(payment.ID.String()),
	)

	return nil
}

// SavePaymentMethod saves the payment method from a successful payment for future autopayments
func (s *PaymentService) SavePaymentMethod(ctx context.Context, customerID int64, paymentMethodID string, planID int64, months int) error {
	return s.customerRepository.SetPaymentMethod(ctx, customerID, paymentMethodID, planID, months)
}

// DisableAutopay disables autopay for a customer
func (s *PaymentService) DisableAutopay(ctx context.Context, telegramID int64) error {
	return s.customerRepository.DisableAutopayByTelegramID(ctx, telegramID)
}

// ProcessAutopayments finds all customers with expiring subscriptions and creates recurring payments
func (s *PaymentService) ProcessAutopayments(ctx context.Context) error {
	if !s.settingsRepository.GetBool("recurring_payments_enabled", false) {
		return nil
	}

	daysBefore := s.settingsRepository.GetInt("recurring_days_before", 3)

	customers, err := s.customerRepository.FindCustomersWithExpiringAutopay(ctx, daysBefore)
	if err != nil {
		return fmt.Errorf("failed to find customers with expiring autopay: %w", err)
	}

	if customers == nil || len(*customers) == 0 {
		return nil
	}

	slog.Info("Processing autopayments", "customer_count", len(*customers))

	maxAttempts := s.settingsRepository.GetInt("recurring_max_failed_attempts", 3)

	for _, customer := range *customers {
		if err := s.CreateRecurringPayment(ctx, &customer); err != nil {
			slog.Error("Failed to create recurring payment",
				"customer_id", utils.MaskHalfInt64(customer.ID),
				"error", err,
			)

			// Increment failed attempts counter
			failedAttempts, incErr := s.customerRepository.IncrementAutopayFailedAttempts(ctx, customer.ID)
			if incErr != nil {
				slog.Error("Failed to increment autopay failed attempts", "customer_id", customer.ID, "error", incErr)
				continue
			}

			// Check if max attempts reached
			if failedAttempts >= maxAttempts {
				// Disable autopay
				if disableErr := s.customerRepository.DisableAutopayAndReset(ctx, customer.ID); disableErr != nil {
					slog.Error("Failed to disable autopay after max attempts", "customer_id", customer.ID, "error", disableErr)
				} else {
					slog.Info("Autopay disabled after max failed attempts", "customer_id", customer.ID, "attempts", failedAttempts)
					// Notify user
					s.notifyAutopayDisabledDueToFailure(ctx, &customer)
				}
			}
			continue
		}

		// Reset failed attempts on success
		if customer.AutopayFailedAttempts > 0 {
			if resetErr := s.customerRepository.ResetAutopayFailedAttempts(ctx, customer.ID); resetErr != nil {
				slog.Error("Failed to reset autopay failed attempts", "customer_id", customer.ID, "error", resetErr)
			}
		}
	}

	return nil
}

// NotifyUpcomingAutopayments sends notifications to customers about upcoming autopayments
func (s *PaymentService) NotifyUpcomingAutopayments(ctx context.Context) error {
	if !s.settingsRepository.GetBool("recurring_payments_enabled", false) {
		return nil
	}

	notifyDays := s.settingsRepository.GetInt("recurring_notify_days_before", 5)
	daysBefore := s.settingsRepository.GetInt("recurring_days_before", 3)

	// Find customers who should be notified but not yet charged
	customers, err := s.customerRepository.FindCustomersWithExpiringAutopay(ctx, notifyDays)
	if err != nil {
		return fmt.Errorf("failed to find customers for notification: %w", err)
	}

	if customers == nil || len(*customers) == 0 {
		return nil
	}

	now := time.Now()
	chargeThreshold := now.AddDate(0, 0, daysBefore)

	for _, customer := range *customers {
		// Skip if already within charging window
		if customer.ExpireAt != nil && customer.ExpireAt.Before(chargeThreshold) {
			continue
		}

		// Send notification with disable button
		_, err := s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    customer.TelegramID,
			ParseMode: models.ParseModeHTML,
			Text:      s.translation.GetText(customer.Language, "autopay_notification"),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: s.translation.GetText(customer.Language, "autopay_disable_button"), CallbackData: "autopay_disable"}},
				},
			},
		})
		if err != nil {
			slog.Error("Failed to send autopay notification",
				"customer_id", utils.MaskHalfInt64(customer.ID),
				"error", err,
			)
		}
	}

	return nil
}

// notifyAutopayDisabledDueToFailure sends a notification to user that autopay was disabled due to failed attempts
func (s *PaymentService) notifyAutopayDisabledDueToFailure(ctx context.Context, customer *database.Customer) {
	_, err := s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    customer.TelegramID,
		ParseMode: models.ParseModeHTML,
		Text:      s.translation.GetText(customer.Language, "autopay_disabled_due_to_failure"),
	})
	if err != nil {
		slog.Error("Failed to send autopay disabled notification",
			"customer_id", utils.MaskHalfInt64(customer.ID),
			"error", err,
		)
	}
}

func (s *PaymentService) processReferralBonus(ctx context.Context, customer *database.Customer, purchaseID int64, purchaseMonths int) error {
	if !s.settingsRepository.GetBool("referral_enabled", true) {
		return nil
	}

	referee, err := s.referralRepository.FindByReferee(ctx, customer.TelegramID)
	if err != nil {
		return err
	}
	if referee == nil {
		return nil
	}

	isFirstBonus := !referee.BonusGranted
	recurringEnabled := s.isRecurringReferralEnabled()

	if !isFirstBonus && !recurringEnabled {
		return nil
	}

	referrerCustomer, err := s.customerRepository.FindByTelegramId(ctx, referee.ReferrerID)
	if err != nil {
		return err
	}
	if referrerCustomer == nil {
		return nil
	}

	var bonusDays int
	if isFirstBonus {
		bonusDays = s.getReferralBonusDays(ctx, referrerCustomer.TelegramID)
	} else if recurringEnabled {
		purchaseDays := purchaseMonths * config.DaysInMonth()
		bonusDays = (purchaseDays * s.getRecurringReferralPercent()) / 100
		if bonusDays < 1 {
			bonusDays = 1
		}
	}

	if bonusDays > 0 {
		referrerUser, err := s.remnawaveClient.CreateOrUpdateUser(ctx, referrerCustomer.ID, referrerCustomer.TelegramID, config.TrafficLimit(), bonusDays, false)
		if err != nil {
			return err
		}

		referrerFieldsToUpdate := map[string]interface{}{
			"subscription_link": referrerUser.GetSubscriptionUrl(),
			"expire_at":         referrerUser.GetExpireAt(),
		}
		err = s.customerRepository.UpdateFields(ctx, referrerCustomer.ID, referrerFieldsToUpdate)
		if err != nil {
			return err
		}

		if isFirstBonus {
			err = s.referralRepository.MarkBonusGranted(ctx, referee.ID)
			if err != nil {
				return err
			}
		}

		slog.Info("Granted referral bonus", "referrer_id", utils.MaskHalfInt64(referrerCustomer.ID), "bonus_days", bonusDays, "is_first", isFirstBonus)

		_, err = s.referralRepository.CreateBonusHistory(ctx, referee.ID, &purchaseID, bonusDays, isFirstBonus)
		if err != nil {
			slog.Error("Error creating referral bonus history", "error", err)
		}

		_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    referrerCustomer.TelegramID,
			ParseMode: models.ParseModeHTML,
			Text:      s.translation.GetText(referrerCustomer.Language, "referral_bonus_granted"),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: s.createConnectKeyboard(referrerCustomer),
			},
		})
		if err != nil {
			slog.Error("Error sending referral bonus message", "error", err)
		}
	}

	return nil
}
