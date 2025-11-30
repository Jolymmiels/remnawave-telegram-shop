package yookasa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/utils"
	"strconv"

	"github.com/google/uuid"
)

type YookasaAPI interface {
	CreatePayment(ctx context.Context, request PaymentRequest, idempotencyKey string) (*Payment, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// getBaseURL returns current YooKassa URL from settings
func (c *Client) getBaseURL() string {
	return config.YookasaUrl()
}

// getAuthHeader returns current auth header from settings
func (c *Client) getAuthHeader() string {
	shopID := config.YookasaShopId()
	secretKey := config.YookasaSecretKey()
	auth := fmt.Sprintf("%s:%s", shopID, secretKey)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	return fmt.Sprintf("Basic %s", encodedAuth)
}

func (c *Client) CreateInvoice(ctx context.Context, amount int, month int, customerId int64, purchaseId int64, returnURL string) (*Payment, error) {
	return c.CreateInvoiceWithOptions(ctx, amount, month, customerId, purchaseId, returnURL, false, nil)
}

// CreateInvoiceWithSavePaymentMethod creates an invoice and saves the payment method for future autopayments
func (c *Client) CreateInvoiceWithSavePaymentMethod(ctx context.Context, amount int, month int, customerId int64, purchaseId int64, returnURL string) (*Payment, error) {
	return c.CreateInvoiceWithOptions(ctx, amount, month, customerId, purchaseId, returnURL, true, nil)
}

// CreateRecurringPayment creates a payment using a saved payment method (for autopayments)
func (c *Client) CreateRecurringPayment(ctx context.Context, amount int, month int, customerId int64, purchaseId int64, paymentMethodID uuid.UUID) (*Payment, error) {
	return c.CreateInvoiceWithOptions(ctx, amount, month, customerId, purchaseId, "", false, &paymentMethodID)
}

// CreateInvoiceWithOptions creates an invoice with optional save_payment_method and payment_method_id
func (c *Client) CreateInvoiceWithOptions(ctx context.Context, amount int, month int, customerId int64, purchaseId int64, returnURL string, savePaymentMethod bool, paymentMethodID *uuid.UUID) (*Payment, error) {
	rub := Amount{
		Value:    strconv.Itoa(amount),
		Currency: "RUB",
	}

	var monthString string
	switch month {
	case 1:
		monthString = "месяц"
	case 3, 4:
		monthString = "месяца"
	default:
		monthString = "месяцев"
	}

	description := fmt.Sprintf("Подписка на %d %s", month, monthString)
	receipt := &Receipt{
		Customer: &Customer{
			Email: config.YookasaEmail(),
		},
		Items: []Item{
			{
				VatCode:        1,
				Quantity:       "1",
				Description:    description,
				Amount:         rub,
				PaymentSubject: "payment",
				PaymentMode:    "full_payment",
			},
		},
	}

	metaData := map[string]any{
		"customerId": customerId,
		"purchaseId": purchaseId,
		"username":   ctx.Value("username"),
	}

	var paymentRequest PaymentRequest

	if paymentMethodID != nil {
		// Recurring payment with saved payment method (no confirmation needed)
		paymentRequest = PaymentRequest{
			Amount:          rub,
			Receipt:         receipt,
			Metadata:        metaData,
			PaymentMethodID: paymentMethodID,
			Capture:         true,
			Description:     description,
		}
	} else {
		// Normal payment with redirect confirmation
		paymentRequest = NewPaymentRequest(
			rub,
			returnURL,
			description,
			receipt,
			metaData,
		)
		paymentRequest.SavePaymentMethod = savePaymentMethod
	}

	idempotencyKey := uuid.New().String()

	payment, err := c.CreatePayment(ctx, paymentRequest, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment, nil
}

func (c *Client) CreatePayment(ctx context.Context, request PaymentRequest, idempotencyKey string) (*Payment, error) {
	cfg := utils.DefaultRetryConfig()
	return utils.WithRetry(ctx, cfg, "yookasa.CreatePayment", func() (*Payment, error) {
		return c.doCreatePayment(ctx, request, idempotencyKey)
	})
}

func (c *Client) doCreatePayment(ctx context.Context, request PaymentRequest, idempotencyKey string) (*Payment, error) {
	paymentURL := fmt.Sprintf("%s/payments", c.getBaseURL())

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", paymentURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.getAuthHeader())
	req.Header.Set("Idempotence-Key", idempotencyKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error while reading invoice resp: %w", err)
		}
		return nil, fmt.Errorf("API return error. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var payment Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &payment, nil
}

func (c *Client) GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	cfg := utils.DefaultRetryConfig()
	return utils.WithRetry(ctx, cfg, "yookasa.GetPayment", func() (*Payment, error) {
		return c.doGetPayment(ctx, paymentID)
	})
}

func (c *Client) doGetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	paymentURL := fmt.Sprintf("%s/payments/%s", c.getBaseURL(), paymentID)

	req, err := http.NewRequestWithContext(ctx, "GET", paymentURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.getAuthHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var payment Payment
		if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &payment, nil
	}

	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
