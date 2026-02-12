package platega

type TransactionRequest struct {
	PaymentMethod  int            `json:"paymentMethod"`
	PaymentDetails PaymentDetails `json:"paymentDetails"`
	Description    string         `json:"description"`
	Return         string         `json:"return"`
	FailedURL      string         `json:"failedUrl"`
	Payload        string         `json:"payload"`
}

type PaymentDetails struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type TransactionResponse struct {
	ID                 string      `json:"id,omitempty"`
	TransactionID      string      `json:"transactionId,omitempty"`
	Status             string      `json:"status"`
	PaymentDetails     interface{} `json:"paymentDetails,omitempty"`
	MerchantName       string      `json:"merchantName,omitempty"`
	MerchantID         string      `json:"merchantId,omitempty"`
	Commission         float64     `json:"comission,omitempty"`
	PaymentMethod      string      `json:"paymentMethod,omitempty"`
	ExpiresIn          string      `json:"expiresIn,omitempty"`
	Return             string      `json:"return,omitempty"`
	CommissionUsdt     float64     `json:"comissionUsdt,omitempty"`
	AmountUsdt         float64     `json:"amountUsdt,omitempty"`
	UsdtRate           float64     `json:"usdtRate,omitempty"`
	QR                 string      `json:"qr,omitempty"`
	PayformSuccessUrl  string      `json:"payformSuccessUrl,omitempty"`
	Payload            string      `json:"payload,omitempty"`
	CommissionType     int         `json:"comissionType,omitempty"`
	ExternalID         string      `json:"externalId,omitempty"`
	Description        string      `json:"description,omitempty"`
	Redirect           string      `json:"redirect,omitempty"`
}

func (t *TransactionResponse) GetTransactionID() string {
	if t.TransactionID != "" {
		return t.TransactionID
	}
	return t.ID
}

const (
	StatusConfirmed    = "CONFIRMED"
	StatusCanceled     = "CANCELED"
	StatusPending      = "PENDING"
	StatusChargebacked = "CHARGEBACKED"
)
