package platega

import (
	"context"
	"fmt"

	"remnawave-tg-shop-bot/internal/database"
)

type Provider struct {
	client *Client
}

func NewProvider(client *Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) InvoiceType() database.InvoiceType {
	return database.InvoiceTypePlatega
}

func (p *Provider) IsConfigured() bool {
	return p.client != nil && p.client.IsConfigured()
}

func (p *Provider) CreateInvoice(
	ctx context.Context,
	purchaseID int64,
	amount float64,
	currency string,
	description string,
	returnURL string,
) (redirectURL string, transactionID string, err error) {
	if p.client == nil {
		return "", "", fmt.Errorf("platega client not configured")
	}

	resp, err := p.client.CreateTransaction(ctx, &CreateTransactionRequest{
		PaymentDetails: PaymentDetails{
			Amount:   amount,
			Currency: currency,
		},
		Description: description,
		Return:      returnURL,
		FailedUrl:   returnURL,
		Payload:     fmt.Sprintf("purchaseId=%d", purchaseID),
	})
	if err != nil {
		return "", "", fmt.Errorf("create platega transaction: %w", err)
	}

	return resp.URL, resp.TransactionId, nil
}
