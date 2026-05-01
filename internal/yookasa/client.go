package yookasa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type YookasaAPI interface {
	CreatePayment(ctx context.Context, request PaymentRequest, idempotencyKey string) (*Payment, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	authHeader string
}

func NewClient(baseURL, shopID, secretKey string) *Client {
	auth := fmt.Sprintf("%s:%s", shopID, secretKey)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		authHeader: fmt.Sprintf("Basic %s", encodedAuth),
	}
}

func (c *Client) CreateInvoice(ctx context.Context, amount int, month int, customerId int64, purchaseId int64) (*Payment, error) {
	rub := Amount{
		Value:    strconv.Itoa(amount),
		Currency: "RUB",
	}

	description := utils.FormatSubscriptionDescription(month)
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
		"username":   remnawave.UsernameFromCtx(ctx),
	}

	paymentRequest := NewPaymentRequest(
		rub,
		config.BotURL(),
		description,
		receipt,
		metaData,
	)

	idempotencyKey := uuid.New().String()

	payment, err := c.CreatePayment(ctx, paymentRequest, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment, nil
}

func (c *Client) CreatePayment(ctx context.Context, request PaymentRequest, idempotencyKey string) (*Payment, error) {
	paymentURL := fmt.Sprintf("%s/payments", c.baseURL)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", paymentURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)
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
	paymentURL := fmt.Sprintf("%s/payments/%s", c.baseURL, paymentID)

	var payment *Payment

	maxRetries := 5
	baseDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", paymentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", c.authHeader)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			payment = new(Payment)
			if err := json.NewDecoder(resp.Body).Decode(payment); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			return payment, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryDelay := baseDelay * time.Duration(1<<attempt)
			log.Printf("Received 429 Too Many Requests. Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("exceeded maximum retries due to 429 Too Many Requests")
}
