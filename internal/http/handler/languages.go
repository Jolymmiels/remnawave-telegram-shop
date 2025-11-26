package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/database"
)

type LanguagesHandler struct {
	customerRepository *database.CustomerRepository
}

func NewLanguagesHandler(customerRepository *database.CustomerRepository) *LanguagesHandler {
	return &LanguagesHandler{customerRepository: customerRepository}
}

func (h *LanguagesHandler) GetLanguages(w http.ResponseWriter, r *http.Request) {
	languages, err := h.customerRepository.GetDistinctLanguages(r.Context())
	if err != nil {
		http.Error(w, "Failed to get languages", http.StatusInternalServerError)
		return
	}

	if languages == nil {
		languages = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}
