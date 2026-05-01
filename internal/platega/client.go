package platega

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultBaseURL = "https://app.platega.io"
	DefaultTimeout = 30 * time.Second
)

type Client struct {
	baseURL    string
	merchantID string
	secret     string
	httpClient *http.Client
}

func NewClient(merchantID, secret string) *Client {
	return &Client{
		baseURL:    DefaultBaseURL,
		merchantID: merchantID,
		secret:     secret,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (c *Client) IsConfigured() bool {
	return c.merchantID != "" && c.secret != ""
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-MerchantId", c.merchantID)
	req.Header.Set("X-Secret", c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*CreateTransactionResponse, error) {
	var resp CreateTransactionResponse
	err := c.doRequest(ctx, http.MethodPost, "/transaction/process", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("create transaction failed: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetTransaction(ctx context.Context, id string) (*TransactionStatusResponse, error) {
	var resp TransactionStatusResponse
	err := c.doRequest(ctx, http.MethodGet, "/transaction/"+url.PathEscape(id), nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("get transaction failed: %w", err)
	}
	return &resp, nil
}
