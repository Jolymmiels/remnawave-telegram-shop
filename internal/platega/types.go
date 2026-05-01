package platega

type PaymentMethod int

const (
	PaymentMethodSBPQR                  PaymentMethod = 2
	PaymentMethodCardsRUB               PaymentMethod = 10
	PaymentMethodCardAcquiring          PaymentMethod = 11
	PaymentMethodInternationalAcquiring PaymentMethod = 12
	PaymentMethodCrypto                 PaymentMethod = 13
)

func (m PaymentMethod) String() string {
	switch m {
	case PaymentMethodSBPQR:
		return "SBPQR"
	case PaymentMethodCardsRUB:
		return "CardsRUB"
	case PaymentMethodCardAcquiring:
		return "CardAcquiring"
	case PaymentMethodInternationalAcquiring:
		return "InternationalAcquiring"
	case PaymentMethodCrypto:
		return "Crypto"
	default:
		return "Unknown"
	}
}

type PaymentStatus string

const (
	StatusPending      PaymentStatus = "PENDING"
	StatusCanceled     PaymentStatus = "CANCELED"
	StatusConfirmed    PaymentStatus = "CONFIRMED"
	StatusChargebacked PaymentStatus = "CHARGEBACKED"
)

func (s PaymentStatus) IsTerminal() bool {
	return s == StatusCanceled || s == StatusConfirmed || s == StatusChargebacked
}

func (s PaymentStatus) IsSuccess() bool {
	return s == StatusConfirmed
}

type PaymentDetails struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type CreateTransactionRequest struct {
	PaymentMethod  PaymentMethod  `json:"paymentMethod"`
	PaymentDetails PaymentDetails `json:"paymentDetails"`
	Description    string         `json:"description"`
	Return         string         `json:"return"`
	FailedUrl      string         `json:"failedUrl"`
	Payload        string         `json:"payload,omitempty"`
}

type CreateTransactionResponse struct {
	PaymentMethod  string        `json:"paymentMethod"`
	TransactionId  string        `json:"transactionId"`
	Redirect       string        `json:"redirect"`
	Return         string        `json:"return"`
	PaymentDetails any           `json:"paymentDetails"`
	Status         PaymentStatus `json:"status"`
	ExpiresIn      string        `json:"expiresIn"`
	MerchantId     string        `json:"merchantId"`
	UsdtRate       float64       `json:"usdtRate,omitempty"`
}

type TransactionStatusResponse struct {
	Id                string         `json:"id"`
	Status            PaymentStatus  `json:"status"`
	PaymentDetails    PaymentDetails `json:"paymentDetails"`
	MerchantName      string         `json:"merchantName"`
	MerchantId        string         `json:"mechantId"`
	Commission        float64        `json:"comission"`
	PaymentMethod     string         `json:"paymentMethod"`
	ExpiresIn         string         `json:"expiresIn"`
	Return            string         `json:"return"`
	CommissionUsdt    float64        `json:"comissionUsdt"`
	AmountUsdt        float64        `json:"amountUsdt"`
	QR                string         `json:"qr"`
	PayformSuccessUrl string         `json:"payformSuccessUrl"`
	Payload           string         `json:"payload"`
	CommissionType    int            `json:"comissionType"`
	ExternalId        string         `json:"externalId"`
	Description       string         `json:"description"`
}

type CallbackPayload struct {
	Id            string        `json:"id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	Status        PaymentStatus `json:"status"`
	PaymentMethod int           `json:"paymentMethod"`
	Payload       string        `json:"payload"`
}
