package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
	"strconv"
)

type PlansHandler struct {
	planRepository     *database.PlanRepository
	purchaseRepository *database.PurchaseRepository
}

func NewPlansHandler(planRepository *database.PlanRepository, purchaseRepository *database.PurchaseRepository) *PlansHandler {
	return &PlansHandler{
		planRepository:     planRepository,
		purchaseRepository: purchaseRepository,
	}
}

type PlansResponse struct {
	Plans []database.Plan `json:"plans"`
}

type PlanRequest struct {
	Name              string `json:"name"`
	Price1            int    `json:"price_1"`
	Price3            int    `json:"price_3"`
	Price6            int    `json:"price_6"`
	Price12           int    `json:"price_12"`
	TrafficLimit      int    `json:"traffic_limit"`
	DeviceLimit       *int   `json:"device_limit"`
	InternalSquads    string `json:"internal_squads"`
	ExternalSquadUUID string `json:"external_squad_uuid"`
	RemnawaveTag      string `json:"remnawave_tag"`
	TributeURL        string `json:"tribute_url"`
	IsActive          bool   `json:"is_active"`
}

// List returns all plans
func (h *PlansHandler) List(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planRepository.FindAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch plans", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PlansResponse{Plans: plans})
}

// Create creates a new plan
func (h *PlansHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	plan := &database.Plan{
		Name:              req.Name,
		Price1:            req.Price1,
		Price3:            req.Price3,
		Price6:            req.Price6,
		Price12:           req.Price12,
		TrafficLimit:      req.TrafficLimit,
		DeviceLimit:       req.DeviceLimit,
		InternalSquads:    req.InternalSquads,
		ExternalSquadUUID: req.ExternalSquadUUID,
		RemnawaveTag:      req.RemnawaveTag,
		TributeURL:        req.TributeURL,
		IsActive:          req.IsActive,
		IsDefault:         false,
	}

	created, err := h.planRepository.Create(r.Context(), plan)
	if err != nil {
		http.Error(w, "Failed to create plan", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// Update updates an existing plan
func (h *PlansHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	plan := &database.Plan{
		ID:                id,
		Name:              req.Name,
		Price1:            req.Price1,
		Price3:            req.Price3,
		Price6:            req.Price6,
		Price12:           req.Price12,
		TrafficLimit:      req.TrafficLimit,
		DeviceLimit:       req.DeviceLimit,
		InternalSquads:    req.InternalSquads,
		ExternalSquadUUID: req.ExternalSquadUUID,
		RemnawaveTag:      req.RemnawaveTag,
		TributeURL:        req.TributeURL,
		IsActive:          req.IsActive,
	}

	updated, err := h.planRepository.Update(r.Context(), plan)
	if err != nil {
		http.Error(w, "Failed to update plan", http.StatusInternalServerError)
		return
	}

	if updated == nil {
		http.Error(w, "Plan not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// Delete deletes a plan
func (h *PlansHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	if err := h.planRepository.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPurchaseCount returns the number of purchases for a plan
func (h *PlansHandler) GetPurchaseCount(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	count, err := h.purchaseRepository.CountByPlanID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to count purchases", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// SetDefault sets a plan as default
func (h *PlansHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	if err := h.planRepository.SetDefault(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
