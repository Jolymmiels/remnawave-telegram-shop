package moynalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrAuth      = errors.New("authentication error")
	ErrRetryable = errors.New("retryable error")
	ErrClient    = errors.New("client error")
)

type Client struct {
	httpClient *http.Client
	baseURL    string

	username string
	password string

	token atomic.Value

	authMu       sync.Mutex
	authInFlight bool
	authCond     *sync.Cond
}

func NewClient(baseURL, username, password string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		username:   username,
		password:   password,
	}
	c.authCond = sync.NewCond(&c.authMu)

	c.token.Store("")

	if err := c.authenticate(); err != nil {
		return nil, fmt.Errorf("initial auth failed: %w", err)
	}

	return c, nil
}

func (c *Client) authenticate() error {
	c.authMu.Lock()

	for c.authInFlight {
		c.authCond.Wait()
	}

	if c.token.Load().(string) != "" {
		c.authMu.Unlock()
		return nil
	}

	c.authInFlight = true
	c.authMu.Unlock()

	err := c.authenticateOnce()

	c.authMu.Lock()
	c.authInFlight = false
	c.authCond.Broadcast()
	c.authMu.Unlock()

	return err
}

func (c *Client) authenticateOnce() error {
	authURL := fmt.Sprintf("%s/auth/lkfl", c.baseURL)

	reqBody, err := json.Marshal(AuthRequest{
		Username: c.username,
		Password: c.password,
		DeviceInfo: DeviceInfo{
			SourceDeviceId: "*",
			SourceType:     "WEB",
			AppVersion:     "1.0.0",
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", ErrAuth, resp.StatusCode, b)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.token.Store(authResp.Token)
	return nil
}

func (c *Client) CreateIncome(amount float64, comment string) (*CreateIncomeResponse, error) {
	const (
		maxRetries     = 3
		baseDelay      = 500 * time.Millisecond
		maxAuthRetries = 1
	)

	var (
		lastErr     error
		authRetries int
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.createIncomeOnce(amount, comment)
		if err == nil {
			return resp, nil
		}

		if errors.Is(err, ErrAuth) {
			if authRetries >= maxAuthRetries {
				return nil, err
			}

			c.token.Store("")

			if err := c.authenticate(); err != nil {
				return nil, fmt.Errorf("reauth failed: %w", err)
			}

			authRetries++
			continue
		}

		if !errors.Is(err, ErrRetryable) {
			return nil, err
		}

		lastErr = err
		time.Sleep(baseDelay * time.Duration(1<<(attempt-1)))
	}

	return nil, fmt.Errorf("create income failed after retries: %w", lastErr)
}

func (c *Client) createIncomeOnce(amount float64, comment string) (*CreateIncomeResponse, error) {
	incomeURL := fmt.Sprintf("%s/income", c.baseURL)
	formattedTime := getFormattedTime()

	reqBody, err := json.Marshal(CreateIncomeRequest{
		OperationTime: parseTimeString(formattedTime),
		RequestTime:   parseTimeString(formattedTime),
		Services: []Service{
			{
				Name:     comment,
				Amount:   amount,
				Quantity: 1,
			},
		},
		TotalAmount: fmt.Sprintf("%.2f", amount),
		Client: IncomeClient{
			IncomeType: "FROM_INDIVIDUAL",
		},
		PaymentType:                     "CASH",
		IgnoreMaxTotalIncomeRestriction: false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", incomeURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	token := c.token.Load().(string)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 YaBrowser/24.12.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return nil, fmt.Errorf("%w: %v", ErrRetryable, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: status %d", ErrAuth, resp.StatusCode)

	case resp.StatusCode >= 500:
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrRetryable, resp.StatusCode, b)

	case resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated:
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrClient, resp.StatusCode, b)
	}

	var incomeResp CreateIncomeResponse
	if err := json.NewDecoder(resp.Body).Decode(&incomeResp); err != nil {
		return nil, err
	}

	return &incomeResp, nil
}

func parseTimeString(timeStr string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05-07:00", timeStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", timeStr)
		if err != nil {
			return time.Now()
		}
	}
	return t
}

func getFormattedTime() string {
	return time.Now().Format("2006-01-02T15:04:05-07:00")
}
