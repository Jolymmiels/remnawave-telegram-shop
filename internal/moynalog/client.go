package moynalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewClient(baseURL, username, password string) (*Client, error) {
	client := &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}

	authResp, err := client.Authenticate(username, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client.token = authResp.Token
	return client, nil
}

func (c *Client) Authenticate(username, password string) (*AuthResponse, error) {
	authURL := fmt.Sprintf("%s/auth/lkfl", c.baseURL)

	deviceInfo := DeviceInfo{
		SourceDeviceId: "*",
		SourceType:     "WEB",
		AppVersion:     "1.0.0",
		MetaDetails: MetaDetails{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 YaBrowser/24.12.0 Safari/537.36",
		},
	}

	authRequest := AuthRequest{
		Username:   username,
		Password:   password,
		DeviceInfo: deviceInfo,
	}

	reqBody, err := json.Marshal(authRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %w", err)
	}

	return &authResp, nil
}

func (c *Client) CreateIncome(amount float64, comment string) (*CreateIncomeResponse, error) {
	incomeURL := fmt.Sprintf("%s/income", c.baseURL)

	formattedTime := getFormattedTime()

	service := Service{
		Name:     comment,
		Amount:   amount,
		Quantity: 1,
	}

	client := IncomeClient{
		ContactPhone: nil,
		DisplayName:  nil,
		INN:          nil,
		IncomeType:   "FROM_INDIVIDUAL",
	}

	incomeRequest := CreateIncomeRequest{
		OperationTime:                   parseTimeString(formattedTime),
		RequestTime:                     parseTimeString(formattedTime),
		Services:                        []Service{service},
		TotalAmount:                     fmt.Sprintf("%.2f", amount),
		Client:                          client,
		PaymentType:                     "CASH",
		IgnoreMaxTotalIncomeRestriction: false,
	}

	reqBody, err := json.Marshal(incomeRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal income request: %w", err)
	}

	req, err := http.NewRequest("POST", incomeURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 YaBrowser/24.12.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}
		return nil, fmt.Errorf("create income failed with status %d: %s", resp.StatusCode, string(body))
	}

	var incomeResp CreateIncomeResponse
	if err := json.NewDecoder(resp.Body).Decode(&incomeResp); err != nil {
		return nil, fmt.Errorf("failed to decode income response: %w", err)
	}

	return &incomeResp, nil
}

func parseTimeString(timeStr string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05-07:00", timeStr)
	if err != nil {
		// Если формат не соответствует, пробуем другой формат
		t, err = time.Parse("2006-01-02T15:04:05", timeStr)
		if err != nil {
			// Если оба формата не подходят, возвращаем текущее время
			return time.Now()
		}
	}
	return t
}

func getFormattedTime() string {
	return time.Now().Format("2006-01-02T15:04:05-07:00")
}
