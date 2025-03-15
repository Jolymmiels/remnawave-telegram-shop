package remnawave

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL string, token string) *Client {
	return &Client{
		token:      token,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (r *Client) CreateOrUpdateUser(ctx context.Context, username string, month int, trafficLimitBytes int64) (*User, error) {
	existingUser, err := r.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	if existingUser == nil {
		newUser, err := r.createUser(ctx, username, month, trafficLimitBytes)
		if err != nil {
			return nil, err
		}
		return newUser, nil
	} else {
		updatedUser, err := r.updateUser(ctx, existingUser, month*30, trafficLimitBytes)
		if err != nil {
			return nil, err
		}
		return updatedUser, nil
	}
}

func (r *Client) updateUser(ctx context.Context, existingUser *User, days int, trafficLimitBytes int64) (*User, error) {
	newExpire := getNewExpire(days, existingUser)

	userUpdate := &UserUpdate{
		UUID:              existingUser.UUID,
		ExpireAt:          newExpire,
		Status:            ACTIVE,
		TrafficLimitBytes: trafficLimitBytes,
	}

	jsonData, err := json.Marshal(userUpdate)
	if err != nil {
		slog.Error("Error while converting to JSON", "error", err)
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/update", r.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Error while creating update request", "error", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("Error while making update request", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Failed to read error response body", "error", err)
		} else {
			bodyString := string(bodyBytes)
			slog.Error("Request failed",
				"status_code", resp.StatusCode,
				"response_body", bodyString)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var wrapper ResponseWrapper[User]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.Response, nil
}

func (r *Client) createUser(ctx context.Context, username string, month int, trafficLimit int64) (*User, error) {
	expireAt := time.Now().UTC().AddDate(0, 0, month*30)

	inbounds := *r.getInbounds(ctx)
	inboundsId := make([]uuid.UUID, len(inbounds))
	for i, inbound := range inbounds {
		inboundsId[i] = inbound.UUID
	}

	userCreate := &UserCreate{
		Username:             username,
		ActiveUserInbounds:   inboundsId,
		Status:               ACTIVE,
		TrafficLimitStrategy: MONTH,
		SubscriptionUuid:     nil,
		ExpireAt:             expireAt,
		TrafficLimitBytes:    trafficLimit,
	}

	jsonData, err := json.Marshal(userCreate)
	if err != nil {
		slog.Error("Error while converting to JSON: %v", err)
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, r.baseURL+"/api/users", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Error while creating create request: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("Error while making create request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Failed to read error response body", err)
		} else {
			bodyString := string(bodyBytes)
			slog.Error("Request failed",
				"status_code", resp.StatusCode,
				"response_body", bodyString)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var wrapper ResponseWrapper[User]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.Response, nil
}

func (r *Client) GetUser(ctx context.Context, username string) (*User, error) {
	url := fmt.Sprintf("%s/api/users/username/%s", r.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var wrapper ResponseWrapper[User]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.Response, nil
}

func getNewExpire(days int, existingUser *User) time.Time {
	if existingUser.ExpireAt.IsZero() {
		return time.Now().UTC().AddDate(0, 0, days)
	}

	return existingUser.ExpireAt.AddDate(0, 0, days)
}

func (r *Client) GetNodes(ctx context.Context) (*[]Node, error) {
	url := fmt.Sprintf("%s/api/nodes/get-all", r.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error("Failed to create request", "error", err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.token)

	slog.Debug("Sending request to get nodes", "url", url)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to execute request", "error", err.Error())
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		bodyString := ""
		if readErr == nil {
			bodyString = string(bodyBytes)
		}

		slog.Error("Request failed",
			"statusCode", resp.StatusCode,
			"responseBody", bodyString)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var wrapper ResponseWrapper[[]Node]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		slog.Error("Failed to decode response", "error", err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.Response, nil
}

func retryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration,
	fn func() error, isRetryable func(error) bool) error {
	var err error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("operation canceled: %w", ctx.Err())
		}

		err = fn()
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt < maxAttempts-1 {
			delay := initialDelay * (1 << attempt)

			slog.Info("Retrying operation",
				"attempt", attempt+1,
				"maxAttempts", maxAttempts,
				"delay", delay,
				"error", err.Error())

			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("operation canceled during retry wait: %w", ctx.Err())
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, err)
}

func isRetryableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if err.Error() == "unexpected status code: 500" ||
		err.Error() == "unexpected status code: 502" ||
		err.Error() == "unexpected status code: 503" ||
		err.Error() == "unexpected status code: 504" {
		return true
	}

	return false
}

func (r *Client) GetNodesWithRetry(ctx context.Context, maxAttempts int, initialDelay time.Duration) (*[]Node, error) {
	var nodes *[]Node

	slog.Info("Fetching all nodes with retry",
		"baseURL", r.baseURL,
		"maxAttempts", maxAttempts,
		"initialDelay", initialDelay.String())

	err := retryWithBackoff(
		ctx,
		maxAttempts,
		initialDelay,
		func() error {
			var err error
			nodes, err = r.GetNodes(ctx)
			return err
		},
		isRetryableError,
	)

	if err != nil {
		slog.Error("Failed to get nodes after retries", "error", err.Error())
		return nil, err
	}

	return nodes, nil
}

func (r *Client) GetNodesWithDefaultRetry(ctx context.Context) (*[]Node, error) {
	return r.GetNodesWithRetry(ctx, 3, 500*time.Millisecond)
}

func (r *Client) getInbounds(ctx context.Context) *[]Inbound {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, config.RemnawaveUrl()+"/api/inbounds", nil)
	if err != nil {
		slog.Error("Error while creating create request: %v", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("Error while making get inbounds request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("status code", resp.StatusCode)
		return nil
	}

	var wrapper ResponseWrapper[[]Inbound]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		slog.Error("Error while decode response: %v", err)
		return nil
	}

	return &wrapper.Response

}
