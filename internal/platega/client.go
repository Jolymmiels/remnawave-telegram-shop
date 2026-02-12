package platega

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient    *http.Client
	baseURL       string
	merchantID    string
	apiKey        string
	returnURL     string
	failedURL     string
	currency      string
	paymentMethod int
}

func NewClient(merchantID, apiKey, returnURL, failedURL, currency string, paymentMethod int) *Client {
	if currency == "" {
		currency = "RUB"
	}
	if paymentMethod == 0 {
		paymentMethod = 2 // СБП по умолчанию
	}
	return &Client{
		httpClient:    &http.Client{},
		baseURL:       "https://app.platega.io",
		merchantID:    merchantID,
		apiKey:        apiKey,
		returnURL:     returnURL,
		failedURL:     failedURL,
		currency:      currency,
		paymentMethod: paymentMethod,
	}
}

func (c *Client) CreateTransaction(ctx context.Context, amount int, description, payload string) (*TransactionResponse, error) {
	reqBody := TransactionRequest{
		PaymentMethod: c.paymentMethod,
		PaymentDetails: PaymentDetails{
			Amount:   amount,
			Currency: c.currency,
		},
		Description: description,
		Return:      c.returnURL,
		FailedURL:   c.failedURL,
		Payload:     payload,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/transaction/process", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MerchantId", c.merchantID)
	req.Header.Set("X-Secret", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned error. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var transactionResp TransactionResponse
	if err := json.Unmarshal(body, &transactionResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if transactionResp.Redirect == "" {
		return nil, fmt.Errorf("API response does not contain redirect URL")
	}

	return &transactionResp, nil
}

func (c *Client) GetMerchantID() string {
	return c.merchantID
}

func (c *Client) GetAPIKey() string {
	return c.apiKey
}

func (c *Client) GetTransaction(ctx context.Context, transactionID string) (*TransactionResponse, error) {
	endpoint := fmt.Sprintf("%s/transaction/%s", c.baseURL, transactionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-MerchantId", c.merchantID)
	req.Header.Set("X-Secret", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var transactionResp TransactionResponse
	if err := json.Unmarshal(body, &transactionResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &transactionResp, nil
}
