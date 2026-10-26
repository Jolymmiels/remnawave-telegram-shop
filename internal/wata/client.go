package wata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.wata.pro/api/h2h"
	DefaultTimeout = 30 * time.Second
)

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func NewClient(baseURL, accessToken string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (c *Client) IsConfigured() bool {
	return c.accessToken != ""
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
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

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

func (c *Client) CreateLink(ctx context.Context, req *CreateLinkRequest) (*PaymentLink, error) {
	var resp PaymentLink
	err := c.doRequest(ctx, http.MethodPost, "/links", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("create payment link failed: %w", err)
	}
	if resp.URL == "" {
		return nil, fmt.Errorf("create payment link failed: empty url in response")
	}
	return &resp, nil
}

// GetPublicKey returns the PEM-encoded key WATA uses to sign webhooks.
func (c *Client) GetPublicKey(ctx context.Context) (string, error) {
	var resp PublicKeyResponse
	err := c.doRequest(ctx, http.MethodGet, "/public-key", nil, &resp)
	if err != nil {
		return "", fmt.Errorf("get public key failed: %w", err)
	}
	return resp.Value, nil
}
