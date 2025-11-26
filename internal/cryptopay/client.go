package cryptopay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CryptoPayApi interface {
	CreateInvoice(invoiceReq *InvoiceRequest) (*InvoiceResponse, error)
	GetInvoices(status, fiat, asset, invoiceIds string, offset, limit int) (*[]InvoiceResponse, error)
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewCryptoPayClient(url string, tokn string) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    url,
		token:      tokn,
	}
}

func (c *Client) CreateInvoice(invoiceReq *InvoiceRequest) (*InvoiceResponse, error) {
	jsonData, err := json.Marshal(invoiceReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling invoice: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/createInvoice", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error while creating invoice req: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Crypto-Pay-API-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while making invoice req: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading invoice resp: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API return error. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var apiResp ResponseWrapper[InvoiceResponse]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error while unmarshiling response: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("API create failed: ok=%v", apiResp.Ok)
	}

	return &apiResp.Result, nil
}

func (c *Client) GetInvoices(status, fiat, asset, invoiceIds string, offset, limit int) (*[]InvoiceResponse, error) {
	endpoint := fmt.Sprintf("%s/api/getInvoices", c.baseURL)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error while creating request: %w", err)
	}

	q := req.URL.Query()

	if status != "" {
		q.Add("status", status)
	}

	if offset > 0 {
		q.Add("offset", fmt.Sprintf("%d", offset))
	}

	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}

	if invoiceIds != "" {
		q.Add("invoice_ids", invoiceIds)
	}

	if fiat != "" {
		q.Add("fiat", fiat)
	}

	if asset != "" {
		q.Add("asset", asset)
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("Crypto-Pay-API-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while making query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var apiResp ResponseListWrapper[InvoiceResponse]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error while unmarshaling json: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("API get invoices failed: %v", apiResp.Ok)
	}

	return &apiResp.Result.Items, nil
}
