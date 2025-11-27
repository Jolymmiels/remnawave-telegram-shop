package tribute

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"strings"
	"time"
)

type Client struct {
	paymentService     *payment.PaymentService
	customerRepository *database.CustomerRepository
}

const (
	CancelledSubscription = "cancelled_subscription"
	NewSubscription       = "new_subscription"
	TestHook              = ""
)

func NewClient(paymentService *payment.PaymentService, customerRepository *database.CustomerRepository) *Client {
	return &Client{
		paymentService:     paymentService,
		customerRepository: customerRepository,
	}
}

func (c *Client) WebHookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*60)
		defer cancel()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("webhook: read body error", "error", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		signature := r.Header.Get("trbt-signature")
		if signature == "" {
			http.Error(w, "missing signature", http.StatusUnauthorized)
			return
		}

		secret := config.GetTributeAPIKey()
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expected), []byte(signature)) {
			log.Printf("webhook: bad signature (expected %s)", expected)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var wh SubscriptionWebhook
		if err := json.Unmarshal(body, &wh); err != nil {
			slog.Error("webhook: unmarshal error", "error", err, "payload", string(body))
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		switch wh.Name {
		case NewSubscription:
			err := c.newSubscriptionHandler(ctx, wh)
			if err != nil {
				slog.Error("webhook: new subscription error", "error", err, "payload", string(body))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case CancelledSubscription:
			err := c.cancelSubscriptionHandler(ctx, wh)
			if errors.Is(err, payment.ErrCustomerNotFound) {
				slog.Warn("webhook: customer not found", "telegram_id", wh.Payload.TelegramUserID)
				w.WriteHeader(http.StatusOK)
				return
			}
			if err != nil {
				slog.Error("webhook: cancel subscription error", "error", err, "payload", string(body))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case TestHook:
			slog.Info("Tribute webhook working")
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (c *Client) cancelSubscriptionHandler(ctx context.Context, wh SubscriptionWebhook) error {
	return c.paymentService.CancelTributePurchase(ctx, wh.Payload.TelegramUserID)
}

func (c *Client) newSubscriptionHandler(ctx context.Context, wh SubscriptionWebhook) error {
	months := convertPeriodToMonths(wh.Payload.Period)

	customer, err := c.customerRepository.FindByTelegramId(ctx, wh.Payload.TelegramUserID)
	_, purchaseId, err := c.paymentService.CreatePurchase(ctx, float64(wh.Payload.Amount), months, customer, database.InvoiceTypeTribute, nil)

	if err != nil {
		return err
	}

	err = c.paymentService.ProcessPurchaseById(ctx, purchaseId)
	if err != nil {
		return err
	}
	return nil
}

func convertPeriodToMonths(period string) int {
	switch strings.ToLower(period) {
	case "monthly":
		return 1
	case "quarterly", "3-month", "3months", "3-months", "q":
		return 3
	case "halfyearly":
		return 6
	case "yearly", "annual", "y":
		return 12
	default:
		return 1
	}
}
