package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/promo"
	"strconv"
	"strings"
	"time"
)

type PromoHandler struct {
	promoService *promo.Service
}

func NewPromoHandler(service *promo.Service) *PromoHandler {
	return &PromoHandler{promoService: service}
}

type CreatePromoRequest struct {
	Code      string     `json:"code"`
	BonusDays int        `json:"bonus_days"`
	MaxUses   *int       `json:"max_uses"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type PromoResponse struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	BonusDays int        `json:"bonus_days"`
	MaxUses   *int       `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
}

func (ph *PromoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	promos, err := ph.promoService.GetAll(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response format
	response := make([]PromoResponse, len(promos))
	for i, promo := range promos {
		response[i] = PromoResponse{
			ID:        promo.ID,
			Code:      promo.Code,
			BonusDays: promo.BonusDays,
			MaxUses:   promo.MaxUses,
			UsedCount: promo.UsedCount,
			ExpiresAt: promo.ExpiresAt,
			Active:    promo.Active,
			CreatedAt: promo.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (ph *PromoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreatePromoRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))

	if req.Code == "" {
		writeErr(w, http.StatusBadRequest, "code is required")
		return
	}
	if req.BonusDays <= 0 {
		writeErr(w, http.StatusBadRequest, "bonus_days must be greater than 0")
		return
	}
	if req.MaxUses != nil && *req.MaxUses <= 0 {
		writeErr(w, http.StatusBadRequest, "max_uses must be greater than 0 if specified")
		return
	}

	createReq := &database.CreatePromoRequest{
		Code:      req.Code,
		BonusDays: req.BonusDays,
		MaxUses:   req.MaxUses,
		ExpiresAt: req.ExpiresAt,
	}

	created, err := ph.promoService.Create(ctx, createReq)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := PromoResponse{
		ID:        created.ID,
		Code:      created.Code,
		BonusDays: created.BonusDays,
		MaxUses:   created.MaxUses,
		UsedCount: created.UsedCount,
		ExpiresAt: created.ExpiresAt,
		Active:    created.Active,
		CreatedAt: created.CreatedAt,
	}

	w.Header().Set("Location", "/api/promos/"+strconv.FormatInt(created.ID, 10))
	writeJSON(w, http.StatusCreated, response)
}

func (ph *PromoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	err = ph.promoService.Update(ctx, id, req.Active)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (ph *PromoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	err = ph.promoService.Delete(ctx, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (ph *PromoHandler) GetUsages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	page := 1
	limit := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	result, err := ph.promoService.GetPromoUsages(ctx, id, page, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (ph *PromoHandler) GetUserPromos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	telegramIDStr := r.PathValue("telegramID")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid telegram_id")
		return
	}

	usages, err := ph.promoService.GetCustomerPromoUsages(ctx, telegramID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, usages)
}
