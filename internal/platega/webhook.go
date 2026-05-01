package platega

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"remnawave-tg-shop-bot/internal/database"
)

type PurchaseProcessor interface {
	ProcessPurchaseById(ctx context.Context, purchaseId int64) error
}

type WebhookHandler struct {
	purchaseRepo       *database.PurchaseRepository
	processor          PurchaseProcessor
	expectedMerchantID string
	expectedSecret     string
}

func NewWebhookHandler(
	purchaseRepo *database.PurchaseRepository,
	processor PurchaseProcessor,
	merchantID, secret string,
) *WebhookHandler {
	return &WebhookHandler{
		purchaseRepo:       purchaseRepo,
		processor:          processor,
		expectedMerchantID: merchantID,
		expectedSecret:     secret,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if h.expectedMerchantID == "" || h.expectedSecret == "" {
		http.Error(w, "platega not configured", http.StatusServiceUnavailable)
		return
	}

	merchantID := r.Header.Get("X-MerchantId")
	secretHeader := r.Header.Get("X-Secret")
	if merchantID != h.expectedMerchantID || secretHeader != h.expectedSecret {
		slog.Warn("platega webhook: invalid credentials", "received_merchant_id", merchantID)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("platega webhook: read body error", "error", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var payload CallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("platega webhook: unmarshal error", "error", err, "payload", string(body))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	slog.Info("platega webhook received",
		"transaction_id", payload.Id,
		"status", payload.Status,
		"amount", payload.Amount,
		"payment_method", payload.PaymentMethod,
	)

	purchaseID, ok := parsePurchaseID(payload.Payload)
	if !ok {
		slog.Warn("platega webhook: missing purchaseId in payload", "transaction_id", payload.Id, "payload", payload.Payload)
		w.WriteHeader(http.StatusOK)
		return
	}

	purchase, err := h.purchaseRepo.FindById(ctx, purchaseID)
	if err != nil {
		slog.Error("platega webhook: find purchase failed", "purchase_id", purchaseID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if purchase == nil {
		slog.Warn("platega webhook: purchase not found", "purchase_id", purchaseID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if purchase.Status == database.PurchaseStatusPaid || purchase.Status == database.PurchaseStatusCancel {
		slog.Info("platega webhook: purchase already finalized", "purchase_id", purchaseID, "status", purchase.Status)
		w.WriteHeader(http.StatusOK)
		return
	}

	switch payload.Status {
	case StatusConfirmed:
		if err := h.processor.ProcessPurchaseById(ctx, purchaseID); err != nil {
			slog.Error("platega webhook: process purchase failed", "purchase_id", purchaseID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case StatusCanceled, StatusChargebacked:
		err := h.purchaseRepo.UpdateFields(ctx, purchaseID, map[string]interface{}{
			"status": database.PurchaseStatusCancel,
		})
		if err != nil {
			slog.Error("platega webhook: cancel purchase failed", "purchase_id", purchaseID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	default:
		slog.Debug("platega webhook: status ignored", "purchase_id", purchaseID, "status", payload.Status)
	}

	w.WriteHeader(http.StatusOK)
}

func parsePurchaseID(payload string) (int64, bool) {
	for _, part := range strings.Split(payload, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "purchaseId" {
			id, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return 0, false
			}
			return id, true
		}
	}
	return 0, false
}
