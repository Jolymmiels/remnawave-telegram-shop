package wata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateLinkSendsDocumentedRequest(t *testing.T) {
	var (
		gotMethod, gotPath, gotAuth string
		gotBody                     map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","status":"Opened","url":"https://payment.wata.pro/pay-form/3fa85f64","orderId":"42","currency":"RUB","amount":99.00}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/api/h2h/", "token")
	link, err := client.CreateLink(context.Background(), &CreateLinkRequest{
		Amount:             99,
		Currency:           "RUB",
		Description:        "Подписка на 1 месяц",
		OrderID:            "42",
		SuccessRedirectURL: "https://t.me/bot",
		FailRedirectURL:    "https://t.me/bot",
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/h2h/links" {
		t.Fatalf("expected POST /api/h2h/links, got %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
	want := map[string]any{
		"amount":             float64(99),
		"currency":           "RUB",
		"description":        "Подписка на 1 месяц",
		"orderId":            "42",
		"successRedirectUrl": "https://t.me/bot",
		"failRedirectUrl":    "https://t.me/bot",
	}
	for key, value := range want {
		if gotBody[key] != value {
			t.Errorf("body[%s] = %v, want %v", key, gotBody[key], value)
		}
	}
	if link.ID != "3fa85f64-5717-4562-b3fc-2c963f66afa6" || link.URL != "https://payment.wata.pro/pay-form/3fa85f64" {
		t.Fatalf("unexpected link: %+v", link)
	}
}

func TestCreateLinkReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"Payment:PL_1001","message":"Your request is not valid!"}}`))
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "token").CreateLink(context.Background(), &CreateLinkRequest{Amount: 1, Currency: "RUB", OrderID: "1"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}
