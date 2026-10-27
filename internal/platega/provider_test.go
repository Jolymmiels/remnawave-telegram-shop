package platega

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateInvoiceUsesLinkWithoutPaymentMethod(t *testing.T) {
	var path string
	var body map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"transactionId":"tx-1","status":"PENDING","url":"https://pay.platega.io/?id=tx-1","expiresIn":"00:15:00","rate":0}`))
	}))
	defer server.Close()

	client := NewClient("merchant", "secret")
	client.baseURL = server.URL
	redirect, txID, err := NewProvider(client).CreateInvoice(context.Background(), 42, 100, "RUB", "test", "https://t.me/bot")
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	if path != "/v2/transaction/process" {
		t.Fatalf("expected /v2/transaction/process, got %s", path)
	}
	if _, ok := body["paymentMethod"]; ok {
		t.Fatalf("paymentMethod must not be sent, the payer picks it on the Platega page")
	}
	if string(body["payload"]) != `"purchaseId=42"` {
		t.Fatalf("expected purchaseId payload for the webhook, got %s", body["payload"])
	}
	if redirect != "https://pay.platega.io/?id=tx-1" || txID != "tx-1" {
		t.Fatalf("expected link and transaction from the v2 response, got %q %q", redirect, txID)
	}
}
