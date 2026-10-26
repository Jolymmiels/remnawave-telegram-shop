package wata

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"remnawave-tg-shop-bot/internal/database"
)

// maxWebhookBodyBytes caps the unauthenticated request body; WATA
// notifications are a few hundred bytes.
const maxWebhookBodyBytes = 1 << 20

// keyRefreshInterval limits how often a bad signature can trigger a public key
// re-fetch, so forged webhooks cannot make the bot hammer the WATA API.
const keyRefreshInterval = time.Minute

type PurchaseFinder interface {
	FindById(ctx context.Context, id int64) (*database.Purchase, error)
}

type PurchaseProcessor interface {
	ProcessPurchaseById(ctx context.Context, purchaseId int64) error
}

type WebhookHandler struct {
	client    *Client
	purchases PurchaseFinder
	processor PurchaseProcessor

	mu           sync.Mutex
	publicKey    *rsa.PublicKey
	keyFetchedAt time.Time
}

func NewWebhookHandler(client *Client, purchases PurchaseFinder, processor PurchaseProcessor) *WebhookHandler {
	return &WebhookHandler{
		client:    client,
		purchases: purchases,
		processor: processor,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err != nil {
		slog.Error("wata webhook: read body error", "error", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.verifySignature(ctx, body, r.Header.Get("X-Signature")); err != nil {
		slog.Warn("wata webhook: invalid signature", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("wata webhook: unmarshal error", "error", err, "payload", string(body))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	slog.Info("wata webhook received",
		"transaction_id", payload.TransactionID,
		"kind", payload.Kind,
		"status", payload.TransactionStatus,
		"amount", payload.Amount,
		"order_id", payload.OrderID,
	)

	// A declined attempt does not close a payment link: the customer can retry
	// on the same form, so only a paid payment changes the purchase.
	if payload.Kind != KindPayment || payload.TransactionStatus != StatusPaid {
		slog.Debug("wata webhook: event ignored", "kind", payload.Kind, "status", payload.TransactionStatus, "order_id", payload.OrderID)
		w.WriteHeader(http.StatusOK)
		return
	}

	purchaseID, err := strconv.ParseInt(payload.OrderID, 10, 64)
	if err != nil {
		slog.Warn("wata webhook: unknown orderId", "order_id", payload.OrderID, "transaction_id", payload.TransactionID)
		w.WriteHeader(http.StatusOK)
		return
	}

	purchase, err := h.purchases.FindById(ctx, purchaseID)
	if err != nil {
		slog.Error("wata webhook: find purchase failed", "purchase_id", purchaseID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if purchase == nil || purchase.InvoiceType != database.InvoiceTypeWata {
		slog.Warn("wata webhook: wata purchase not found", "purchase_id", purchaseID, "transaction_id", payload.TransactionID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if purchase.Status == database.PurchaseStatusPaid {
		slog.Info("wata webhook: purchase already paid", "purchase_id", purchaseID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if !strings.EqualFold(payload.Currency, purchase.Currency) || payload.Amount+0.005 < purchase.Amount {
		slog.Error("wata webhook: payment does not cover purchase",
			"purchase_id", purchaseID,
			"amount", payload.Amount, "currency", payload.Currency,
			"expected_amount", purchase.Amount, "expected_currency", purchase.Currency,
		)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.processor.ProcessPurchaseById(ctx, purchaseID); err != nil {
		slog.Error("wata webhook: process purchase failed", "purchase_id", purchaseID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// verifySignature checks the SHA512withRSA signature WATA puts in X-Signature
// over the raw request body.
func (h *WebhookHandler) verifySignature(ctx context.Context, body []byte, signature string) error {
	if signature == "" {
		return errors.New("missing X-Signature header")
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	digest := sha512.Sum512(body)

	key, err := h.getPublicKey(ctx, false)
	if err != nil {
		return err
	}
	verifyErr := rsa.VerifyPKCS1v15(key, crypto.SHA512, digest[:], sig)
	if verifyErr == nil {
		return nil
	}

	// WATA may have rotated its key since it was cached.
	refreshed, err := h.getPublicKey(ctx, true)
	if err != nil {
		return err
	}
	if refreshed == key {
		return verifyErr
	}
	return rsa.VerifyPKCS1v15(refreshed, crypto.SHA512, digest[:], sig)
}

func (h *WebhookHandler) getPublicKey(ctx context.Context, refresh bool) (*rsa.PublicKey, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.publicKey != nil && (!refresh || time.Since(h.keyFetchedAt) < keyRefreshInterval) {
		return h.publicKey, nil
	}

	value, err := h.client.GetPublicKey(ctx)
	if err != nil {
		return nil, err
	}
	key, err := parsePublicKey(value)
	if err != nil {
		return nil, err
	}
	h.publicKey = key
	h.keyFetchedAt = time.Now()
	return key, nil
}

func parsePublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("wata public key: no PEM block")
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("wata public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("wata public key: unexpected key type %T", parsed)
	}
	return key, nil
}
