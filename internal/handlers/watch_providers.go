package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// WatchProviderHandler holds dependencies for watch provider endpoints.
type WatchProviderHandler struct {
	DB *db.DB
}

type watchProviderDTO struct {
	ProviderID      int    `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	LogoPath        string `json:"logo_path"`
	DisplayPriority int    `json:"display_priority"`
}

func toWatchProviderDTO(p *db.WatchProvider) watchProviderDTO {
	return watchProviderDTO{
		ProviderID:      p.ProviderID,
		ProviderName:    p.ProviderName,
		LogoPath:        p.LogoPath,
		DisplayPriority: p.DisplayPriority,
	}
}

// HandleWatchProviders handles GET /api/watch-providers (list all available providers).
func (h *WatchProviderHandler) HandleWatchProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	providers, err := h.DB.GetWatchProviders()
	if err != nil {
		slog.Error("failed to list watch providers", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to list watch providers")
		return
	}

	dtos := make([]watchProviderDTO, 0, len(providers))
	for i := range providers {
		dtos = append(dtos, toWatchProviderDTO(&providers[i]))
	}
	writeJSON(w, http.StatusOK, dtos)
}

type setUserWatchProvidersRequest struct {
	ProviderIDs []int `json:"provider_ids"`
}

// HandleUserWatchProviders handles GET and PUT on /api/user/watch-providers.
func (h *WatchProviderHandler) HandleUserWatchProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getUserWatchProviders(w, r)
	case http.MethodPut:
		h.setUserWatchProviders(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *WatchProviderHandler) getUserWatchProviders(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	providers, err := h.DB.GetUserWatchProviders(userID)
	if err != nil {
		slog.Error("failed to get user watch providers", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get watch providers")
		return
	}

	dtos := make([]watchProviderDTO, 0, len(providers))
	for i := range providers {
		dtos = append(dtos, toWatchProviderDTO(&providers[i]))
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (h *WatchProviderHandler) setUserWatchProviders(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req setUserWatchProvidersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Deduplicate provider IDs
	seen := make(map[int]bool, len(req.ProviderIDs))
	unique := make([]int, 0, len(req.ProviderIDs))
	for _, id := range req.ProviderIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	// Validate that all provider IDs exist
	if len(unique) > 0 {
		invalid, err := h.DB.ValidateProviderIDs(unique)
		if err != nil {
			slog.Error("failed to validate provider IDs", slog.Any("error", err))
			writeError(w, http.StatusInternalServerError, "failed to validate provider IDs")
			return
		}
		if len(invalid) > 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid provider IDs: %v", invalid))
			return
		}
	}

	if err := h.DB.SetUserWatchProviders(userID, unique); err != nil {
		slog.Error("failed to set user watch providers", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to save watch providers")
		return
	}

	// Return the updated list
	providers, err := h.DB.GetUserWatchProviders(userID)
	if err != nil {
		slog.Error("failed to get user watch providers after save", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "saved but failed to retrieve watch providers")
		return
	}

	dtos := make([]watchProviderDTO, 0, len(providers))
	for i := range providers {
		dtos = append(dtos, toWatchProviderDTO(&providers[i]))
	}
	writeJSON(w, http.StatusOK, dtos)
}
