package platega

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"remnawave-tg-shop-bot/internal/database"
)

// documentedPaymentMethods mirrors the PaymentMethodInt enum published at
// https://docs.platega.io/paymentmethodint-13226216d0.
var documentedPaymentMethods = map[int]bool{2: true, 3: true, 11: true, 12: true, 13: true, 14: true}

func TestProviderForRejectsRetiredCardsMethod(t *testing.T) {
	provider, err := ProviderFor(NewClient("merchant", "secret"), database.InvoiceTypePlategaCards)
	if err == nil {
		t.Fatalf("expected retired %s invoice type to be rejected, got provider with method %d", database.InvoiceTypePlategaCards, provider.paymentMethod)
	}
}

func TestProvidersSendDocumentedPaymentMethods(t *testing.T) {
	cases := map[database.InvoiceType]int{
		database.InvoiceTypePlategaSBP:       2,
		database.InvoiceTypePlategaAcquiring: 11,
		database.InvoiceTypePlategaWorldwide: 12,
		database.InvoiceTypePlategaCrypto:    13,
	}
	if len(invoiceTypeToMethod) != len(cases) {
		t.Fatalf("expected %d routable invoice types, got %d", len(cases), len(invoiceTypeToMethod))
	}

	for invoiceType, want := range cases {
		t.Run(string(invoiceType), func(t *testing.T) {
			var body struct {
				PaymentMethod json.RawMessage `json:"paymentMethod"`
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				w.Write([]byte(`{"transactionId":"tx-1","redirect":"https://pay.test/tx-1"}`))
			}))
			defer server.Close()

			client := NewClient("merchant", "secret")
			client.baseURL = server.URL
			provider, err := ProviderFor(client, invoiceType)
			if err != nil {
				t.Fatalf("provider for %s: %v", invoiceType, err)
			}
			if _, _, err := provider.CreateInvoice(context.Background(), 1, 100, "RUB", "test", "https://t.me/bot"); err != nil {
				t.Fatalf("create invoice: %v", err)
			}

			var got int
			if err := json.Unmarshal(body.PaymentMethod, &got); err != nil {
				t.Fatalf("paymentMethod must be an integer, got %s", body.PaymentMethod)
			}
			if got != want || !documentedPaymentMethods[got] {
				t.Fatalf("expected documented paymentMethod %d, got %d", want, got)
			}
		})
	}
}
