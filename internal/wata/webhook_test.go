package wata

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"remnawave-tg-shop-bot/internal/database"
)

// wataPublicKey is the key served by https://api.wata.pro/api/h2h/public-key.
const wataPublicKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzFGBL6kjFth0EfWnYwTv\nxIt4yk4e6tVIB95Zylkepf4586waRtVUPp+3YxRJkx4/eDawHlKjK9NnM72ycl52\nsPklcgGOHK/TbMnKrxQXyy19pzcoTzwzw7cXYSf3AjQTctr39M5iyZo6CZeY1H2Z\nF2LrC9lYTsM8EHsdGGZjsIcDkrwGDw8i/kIIO1AgZ73IdoLlsvaWICeZBc0vhjxZ\nNsjTVNemmDoTQownb5tY51o1awgJZJjPO7iz00HXLw29da7fcklZH0M4GtNF4odf\nVQrEVtb3HLMNoe+l9r6Mc06XmdPyk3aMDaLizcCULltYScBxTImHmcjgCbO0/WT1\ngQIDAQAB\n-----END PUBLIC KEY-----\n"

type fakePurchases map[int64]*database.Purchase

func (f fakePurchases) FindById(_ context.Context, id int64) (*database.Purchase, error) {
	return f[id], nil
}

type fakeProcessor struct {
	processed []int64
	err       error
}

func (p *fakeProcessor) ProcessPurchaseById(_ context.Context, purchaseId int64) error {
	p.processed = append(p.processed, purchaseId)
	return p.err
}

type keyServer struct {
	server  *httptest.Server
	key     atomic.Pointer[rsa.PrivateKey]
	fetches atomic.Int32
}

func newKeyServer(t *testing.T) *keyServer {
	t.Helper()
	ks := &keyServer{}
	ks.key.Store(newKey(t))
	ks.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public-key" {
			http.NotFound(w, r)
			return
		}
		ks.fetches.Add(1)
		der, err := x509.MarshalPKIXPublicKey(&ks.key.Load().PublicKey)
		if err != nil {
			t.Errorf("marshal key: %v", err)
		}
		json.NewEncoder(w).Encode(PublicKeyResponse{Value: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))})
	}))
	t.Cleanup(ks.server.Close)
	return ks
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func sign(t *testing.T, key *rsa.PrivateKey, body []byte) string {
	t.Helper()
	digest := sha512.Sum512(body)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func pendingWataPurchase() *database.Purchase {
	return &database.Purchase{ID: 42, Amount: 99, Currency: "RUB", Status: database.PurchaseStatusPending, InvoiceType: database.InvoiceTypeWata}
}

// paidWebhook mirrors the documented notification, keeping its original formatting.
const paidWebhook = `{
  "transactionType": "SBP",
  "kind": "Payment",
  "id": "3a1cf611-abc6-8d30-c4cd-521c9f6eeeb0",
  "transactionId": "3a16a4f0-27b0-09d1-16da-ba8d5c63eae3",
  "transactionStatus": "Paid",
  "terminalPublicId": "3b16a2f1-dead-4e5d-abff-90865d1e13b1",
  "errorCode": null,
  "errorDescription": null,
  "terminalName": "string",
  "amount": 99.00,
  "currency": "RUB",
  "orderId": "42",
  "orderDescription": "Подписка на 1 месяц",
  "paymentTime": "2024-12-04T17:41:44.434598Z",
  "commission": 10,
  "email": null,
  "paymentLinkId": "3fa85f64-5717-4562-b3fc-2c963f66afa6"
}`

func serve(handler *WebhookHandler, body []byte, signature string) int {
	req := httptest.NewRequest(http.MethodPost, "/webhook/wata", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-Signature", signature)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

func TestWebhookProcessesSignedPaidPayment(t *testing.T) {
	ks := newKeyServer(t)
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: pendingWataPurchase()}, processor)

	body := []byte(paidWebhook)
	if code := serve(handler, body, sign(t, ks.key.Load(), body)); code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(processor.processed) != 1 || processor.processed[0] != 42 {
		t.Fatalf("expected purchase 42 processed once, got %v", processor.processed)
	}
}

func TestWebhookRejectsUnsignedOrForgedRequests(t *testing.T) {
	ks := newKeyServer(t)
	body := []byte(paidWebhook)
	cases := map[string]string{
		"missing signature": "",
		"foreign key":       sign(t, newKey(t), body),
		"not base64":        "%%%",
		"altered body":      sign(t, ks.key.Load(), bytes.Replace(body, []byte(`"amount": 99.00`), []byte(`"amount": 1.00`), 1)),
	}
	for name, signature := range cases {
		t.Run(name, func(t *testing.T) {
			processor := &fakeProcessor{}
			handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: pendingWataPurchase()}, processor)
			if code := serve(handler, body, signature); code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", code)
			}
			if len(processor.processed) != 0 {
				t.Fatalf("forged webhook processed purchases %v", processor.processed)
			}
		})
	}
}

func TestWebhookRejectsOversizedBodyBeforeVerification(t *testing.T) {
	ks := newKeyServer(t)
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: pendingWataPurchase()}, processor)

	body := bytes.Repeat([]byte(" "), maxWebhookBodyBytes+1)
	if code := serve(handler, body, sign(t, ks.key.Load(), body)); code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an oversized body, got %d", code)
	}
	if ks.fetches.Load() != 0 || len(processor.processed) != 0 {
		t.Fatalf("oversized body reached verification: %d key fetches, processed %v", ks.fetches.Load(), processor.processed)
	}
}

func TestWebhookDoesNotProcessIneligibleEvents(t *testing.T) {
	cases := map[string]struct {
		purchase *database.Purchase
		mutate   func(map[string]any)
	}{
		"declined attempt":  {pendingWataPurchase(), func(p map[string]any) { p["transactionStatus"] = "Declined" }},
		"pre-payment check": {pendingWataPurchase(), func(p map[string]any) { p["transactionStatus"] = "Created" }},
		"refund":            {pendingWataPurchase(), func(p map[string]any) { p["kind"] = "Refund" }},
		"missing kind":      {pendingWataPurchase(), func(p map[string]any) { delete(p, "kind") }},
		"unknown kind":      {pendingWataPurchase(), func(p map[string]any) { p["kind"] = "Payout" }},
		"already paid": {func() *database.Purchase {
			p := pendingWataPurchase()
			p.Status = database.PurchaseStatusPaid
			return p
		}(), func(map[string]any) {}},
		"other provider purchase": {func() *database.Purchase {
			p := pendingWataPurchase()
			p.InvoiceType = database.InvoiceTypePlatega
			return p
		}(), func(map[string]any) {}},
		"unknown order":   {pendingWataPurchase(), func(p map[string]any) { p["orderId"] = "43" }},
		"foreign orderId": {pendingWataPurchase(), func(p map[string]any) { p["orderId"] = "shop-42" }},
		"underpaid":       {pendingWataPurchase(), func(p map[string]any) { p["amount"] = 9.99 }},
		"other currency":  {pendingWataPurchase(), func(p map[string]any) { p["currency"] = "USD" }},
	}

	ks := newKeyServer(t)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var payload map[string]any
			if err := json.Unmarshal([]byte(paidWebhook), &payload); err != nil {
				t.Fatal(err)
			}
			tc.mutate(payload)
			body, _ := json.Marshal(payload)

			processor := &fakeProcessor{}
			handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: tc.purchase}, processor)
			if code := serve(handler, body, sign(t, ks.key.Load(), body)); code != http.StatusOK {
				t.Fatalf("expected 200 so WATA stops retrying, got %d", code)
			}
			if len(processor.processed) != 0 {
				t.Fatalf("expected no processing, got %v", processor.processed)
			}
		})
	}
}

func TestWebhookAsksForRetryWhenProcessingFails(t *testing.T) {
	ks := newKeyServer(t)
	processor := &fakeProcessor{err: errors.New("remnawave down")}
	handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: pendingWataPurchase()}, processor)

	body := []byte(paidWebhook)
	if code := serve(handler, body, sign(t, ks.key.Load(), body)); code != http.StatusInternalServerError {
		t.Fatalf("expected 500 so WATA retries, got %d", code)
	}
}

func TestWebhookCachesPublicKeyAndPicksUpRotation(t *testing.T) {
	ks := newKeyServer(t)
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(NewClient(ks.server.URL, ""), fakePurchases{42: pendingWataPurchase()}, processor)
	body := []byte(paidWebhook)

	serve(handler, body, sign(t, ks.key.Load(), body))
	serve(handler, body, sign(t, ks.key.Load(), body))
	if got := ks.fetches.Load(); got != 1 {
		t.Fatalf("expected the public key to be fetched once, got %d", got)
	}

	// A bad signature right after a fetch must not trigger another fetch.
	serve(handler, body, sign(t, newKey(t), body))
	if got := ks.fetches.Load(); got != 1 {
		t.Fatalf("forged webhook re-fetched the key: %d fetches", got)
	}

	// WATA rotates its key; the cached one is older than the refresh interval.
	ks.key.Store(newKey(t))
	handler.keyFetchedAt = time.Now().Add(-2 * keyRefreshInterval)
	if code := serve(handler, body, sign(t, ks.key.Load(), body)); code != http.StatusOK {
		t.Fatalf("expected rotated key to be accepted, got %d", code)
	}
	if got := ks.fetches.Load(); got != 2 {
		t.Fatalf("expected a single re-fetch after rotation, got %d fetches", got)
	}
	if len(processor.processed) != 3 {
		t.Fatalf("expected 3 processed payments, got %v", processor.processed)
	}
}

func TestParsePublicKeyFormats(t *testing.T) {
	if _, err := parsePublicKey(wataPublicKey); err != nil {
		t.Fatalf("parse WATA production key: %v", err)
	}

	key := newKey(t)
	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)})
	parsed, err := parsePublicKey(string(pkcs1))
	if err != nil {
		t.Fatalf("parse PKCS1 key: %v", err)
	}
	if !parsed.Equal(&key.PublicKey) {
		t.Fatal("parsed PKCS1 key differs")
	}

	if _, err := parsePublicKey("not a key"); err == nil {
		t.Fatal("expected error for garbage key")
	}
}
