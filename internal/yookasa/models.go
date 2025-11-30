package yookasa

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID            uuid.UUID         `json:"id,omitempty"`
	Status        string            `json:"status,omitempty"`
	Paid          bool              `json:"paid,omitempty"`
	Amount        Amount            `json:"amount,omitempty"`
	Confirmation  ConfirmationType  `json:"confirmation,omitempty"`
	CreatedAt     time.Time         `json:"created_at,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at,omitempty"`
	Description   string            `json:"description,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Recipient     RecipientType     `json:"recipient,omitempty"`
	PaymentMethod PaymentType       `json:"payment_method,omitempty"`
	Refundable    bool              `json:"refundable,omitempty"`
	Test          bool              `json:"test,omitempty"`
	RedirectURL   string            `json:"redirect_url,omitempty"`
}

func (p *Payment) IsCancelled() bool {
	return p.Status == "canceled"
}

type PaymentRequest struct {
	Amount            Amount             `json:"amount"`
	Confirmation      *ConfirmationType  `json:"confirmation,omitempty"`
	Capture           bool               `json:"capture"`
	Description       string             `json:"description,omitempty"`
	PaymentMethodData *PaymentMethodData `json:"payment_method_data,omitempty"`
	SavePaymentMethod bool               `json:"save_payment_method"`
	PaymentMethodID   *uuid.UUID         `json:"payment_method_id"`
	Receipt           *Receipt           `json:"receipt,omitempty"`
	Metadata          map[string]any     `json:"metadata,omitempty"`
}

func NewPaymentRequest(
	amount Amount,
	urlRedirect,
	description string,
	receipt *Receipt,
	metadata map[string]any) PaymentRequest {
	return PaymentRequest{
		Amount:   amount,
		Receipt:  receipt,
		Metadata: metadata,
		Confirmation: &ConfirmationType{
			Type:      "redirect",
			ReturnURL: urlRedirect,
		},
		PaymentMethodData: nil,
		Capture:           true,
		Description:       description,
	}
}

type Receipt struct {
	Items    []Item    `json:"items"`
	Customer *Customer `json:"customer,omitempty"`
}

type Customer struct {
	Email string `json:"email"`
}

type Item struct {
	Description    string `json:"description"`
	Amount         Amount `json:"amount"`
	VatCode        int    `json:"vat_code"`
	Quantity       string `json:"quantity"`
	PaymentSubject string `json:"payment_subject,omitempty"`
	PaymentMode    string `json:"payment_mode,omitempty"`
}

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type PaymentMethodData struct {
	Type string `json:"type"`
}

type ConfirmationType struct {
	ReturnURL       string `json:"return_url,omitempty"`
	Type            string `json:"type,omitempty"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

type RecipientType struct {
	AccountID string `json:"account_id,omitempty"`
	GatewayID string `json:"gateway_id,omitempty"`
}

type PaymentType struct {
	Type   string    `json:"type,omitempty"`
	ID     uuid.UUID `json:"id,omitempty"`
	Saved  bool      `json:"saved,omitempty"`
	Status string    `json:"status,omitempty"` // pending, active, inactive
}

// GetSavedPaymentMethodID returns the payment method ID if it was saved, nil otherwise
func (p *Payment) GetSavedPaymentMethodID() *uuid.UUID {
	if p.PaymentMethod.Saved && p.PaymentMethod.ID != uuid.Nil {
		return &p.PaymentMethod.ID
	}
	return nil
}
