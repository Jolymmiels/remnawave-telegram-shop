package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/broadcast"
	"remnawave-tg-shop-bot/internal/database"
	"strconv"
	"strings"
)

type BroadcastHandler struct {
	broadcastService *broadcast.Service
}

func NewBroadcastHandler(service *broadcast.Service) *BroadcastHandler {
	return &BroadcastHandler{broadcastService: service}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (bh *BroadcastHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	typ := strings.TrimSpace(strings.ToLower(q.Get("type")))
	lang := strings.TrimSpace(q.Get("language"))
	status := strings.TrimSpace(strings.ToLower(q.Get("status")))

	limit := 50
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 200 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	sortRaw := q.Get("sort")
	desc := true
	if sortRaw == "created_at" {
		desc = false
	}

	items, err := bh.broadcastService.List(ctx, database.BroadcastListParams{
		Type:     typ,
		Language: lang,
		Status:   status,
		Limit:    limit,
		Offset:   offset,
		SortBy:   "created_at",
		Desc:     desc,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Offset", strconv.Itoa(offset))
	if len(*items) == limit {
		w.Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
	}

	writeJSON(w, http.StatusOK, items)
}

//func (bh *BroadcastHandler) GetByID(w http.ResponseWriter, r *http.Request) {
//	ctx := r.Context()
//	idStr := r.PathValue("id")
//	id, err := strconv.ParseInt(idStr, 10, 64)
//	if err != nil {
//		writeErr(w, http.StatusBadRequest, "invalid id")
//		return
//	}
//
//	br, err := bh.repo.GetByID(ctx, id)
//	if err != nil {
//		if err == pgx.ErrNoRows {
//			writeErr(w, http.StatusNotFound, "not found")
//			return
//		}
//		writeErr(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//
//	writeJSON(w, http.StatusOK, br)
//}

func (bh *BroadcastHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := bh.broadcastService.Delete(ctx, id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (bh *BroadcastHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Content  string `json:"content"`
		Type     string `json:"type"`
		Language string `json:"language"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	req.Language = strings.TrimSpace(req.Language)

	if req.Content == "" {
		writeErr(w, http.StatusBadRequest, "content is required")
		return
	}
	switch req.Type {
	case database.BroadcastAll, database.BroadcastActive, database.BroadcastInactive:
	default:
		writeErr(w, http.StatusBadRequest, "type must be one of: all, active, inactive")
		return
	}

	created, err := bh.broadcastService.CreateBroadcast(ctx, req.Content, req.Type, req.Language)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Location", "/api/broadcasts/"+strconv.FormatInt(created.ID, 10))
	writeJSON(w, http.StatusCreated, created)
}
