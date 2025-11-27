package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
)

type SettingsHandler struct {
	settingsRepository *database.SettingsRepository
	remnawaveClient    *remnawave.Client
}

func NewSettingsHandler(settingsRepository *database.SettingsRepository, remnawaveClient *remnawave.Client) *SettingsHandler {
	return &SettingsHandler{
		settingsRepository: settingsRepository,
		remnawaveClient:    remnawaveClient,
	}
}

type SettingsResponse struct {
	Settings map[string]string `json:"settings"`
}

type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings"`
}

type SquadsResponse struct {
	InternalSquads []remnawave.Squad `json:"internal_squads"`
	ExternalSquads []remnawave.Squad `json:"external_squads"`
}

// GetSettings returns all settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	settings := h.settingsRepository.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SettingsResponse{Settings: settings})
}

// UpdateSettings updates multiple settings
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.settingsRepository.SetMultiple(r.Context(), req.Settings); err != nil {
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetSquads returns all available squads from Remnawave
func (h *SettingsHandler) GetSquads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	internalSquads, err := h.remnawaveClient.GetSquads(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch internal squads", http.StatusInternalServerError)
		return
	}

	externalSquads, err := h.remnawaveClient.GetExternalSquads(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch external squads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SquadsResponse{
		InternalSquads: internalSquads,
		ExternalSquads: externalSquads,
	})
}
