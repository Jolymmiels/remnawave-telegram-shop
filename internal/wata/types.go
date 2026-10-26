package wata

type TransactionStatus string

const (
	StatusCreated  TransactionStatus = "Created"
	StatusPending  TransactionStatus = "Pending"
	StatusPaid     TransactionStatus = "Paid"
	StatusDeclined TransactionStatus = "Declined"
)

type TransactionKind string

const (
	KindPayment TransactionKind = "Payment"
	KindRefund  TransactionKind = "Refund"
)

type CreateLinkRequest struct {
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency"`
	Description        string  `json:"description,omitempty"`
	OrderID            string  `json:"orderId"`
	SuccessRedirectURL string  `json:"successRedirectUrl,omitempty"`
	FailRedirectURL    string  `json:"failRedirectUrl,omitempty"`
}

type PaymentLink struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	OrderID  string `json:"orderId"`
	Currency string `json:"currency"`
}

type PublicKeyResponse struct {
	Value string `json:"value"`
}

type WebhookPayload struct {
	TransactionType   string            `json:"transactionType"`
	Kind              TransactionKind   `json:"kind"`
	ID                string            `json:"id"`
	TransactionID     string            `json:"transactionId"`
	TransactionStatus TransactionStatus `json:"transactionStatus"`
	ErrorCode         *string           `json:"errorCode"`
	ErrorDescription  *string           `json:"errorDescription"`
	Amount            float64           `json:"amount"`
	Currency          string            `json:"currency"`
	OrderID           string            `json:"orderId"`
	PaymentLinkID     *string           `json:"paymentLinkId"`
}
