package yookasa

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"

	"github.com/google/uuid"
)

type PurchaseProcessor interface {
	ProcessPurchaseById(ctx context.Context, purchaseId int64) error
	CancelYookassaPayment(purchaseId int64) error
}

type WebhookHandler struct {
	client       *Client
	processor    PurchaseProcessor
	purchaseRepo *database.PurchaseRepository
}

func NewWebhookHandler(client *Client, processor PurchaseProcessor, purchaseRepo *database.PurchaseRepository) *WebhookHandler {
	return &WebhookHandler{
		client:       client,
		processor:    processor,
		purchaseRepo: purchaseRepo,
	}
}

type webhookEnvelope struct {
	Type   string `json:"type"`
	Event  string `json:"event"`
	Object struct {
		ID       uuid.UUID         `json:"id"`
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
	} `json:"object"`
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("yookassa webhook: read body error", "error", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var env webhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		slog.Error("yookassa webhook: unmarshal error", "error", err, "payload", string(body))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	slog.Info("yookassa webhook received", "event", env.Event, "payment_id", env.Object.ID)

	if env.Event != "payment.succeeded" && env.Event != "payment.canceled" {
		slog.Debug("yookassa webhook: ignored event", "event", env.Event)
		w.WriteHeader(http.StatusOK)
		return
	}

	payment, err := h.client.GetPayment(ctx, env.Object.ID)
	if err != nil {
		slog.Error("yookassa webhook: get payment failed", "payment_id", env.Object.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	purchaseIDStr, ok := payment.Metadata["purchaseId"]
	if !ok {
		slog.Warn("yookassa webhook: missing purchaseId in metadata", "payment_id", env.Object.ID)
		w.WriteHeader(http.StatusOK)
		return
	}
	purchaseID, err := strconv.ParseInt(purchaseIDStr, 10, 64)
	if err != nil {
		slog.Error("yookassa webhook: invalid purchaseId", "purchaseId", purchaseIDStr, "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	purchase, err := h.purchaseRepo.FindById(ctx, purchaseID)
	if err != nil {
		slog.Error("yookassa webhook: find purchase failed", "purchase_id", purchaseID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if purchase == nil {
		slog.Warn("yookassa webhook: purchase not found", "purchase_id", purchaseID)
		w.WriteHeader(http.StatusOK)
		return
	}
	if purchase.Status == database.PurchaseStatusPaid || purchase.Status == database.PurchaseStatusCancel {
		slog.Info("yookassa webhook: purchase already finalized", "purchase_id", purchaseID, "status", purchase.Status)
		w.WriteHeader(http.StatusOK)
		return
	}

	if payment.IsCancelled() {
		if err := h.processor.CancelYookassaPayment(purchaseID); err != nil {
			slog.Error("yookassa webhook: cancel failed", "purchase_id", purchaseID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if !payment.Paid {
		slog.Info("yookassa webhook: payment not paid yet", "purchase_id", purchaseID, "status", payment.Status)
		w.WriteHeader(http.StatusOK)
		return
	}

	ctxWithUsername := context.WithValue(ctx, remnawave.CtxKeyUsername, payment.Metadata["username"])
	if err := h.processor.ProcessPurchaseById(ctxWithUsername, purchaseID); err != nil {
		slog.Error("yookassa webhook: process purchase failed", "purchase_id", purchaseID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
