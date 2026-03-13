package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// ArrServiceHandler holds dependencies for *arr service endpoints.
type ArrServiceHandler struct {
	DB *db.DB
}

type arrServiceRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

type arrServiceDTO struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	URL       string       `json:"url"`
	APIKey    string       `json:"api_key"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

func toArrServiceDTO(s *db.ArrService) arrServiceDTO {
	return arrServiceDTO{
		ID:        s.ID,
		Name:      s.Name,
		Type:      string(s.Type),
		URL:       s.URL,
		APIKey:    s.APIKey,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

var validArrTypes = map[string]db.ArrServiceType{
	"radarr":   db.ArrServiceTypeRadarr,
	"sonarr":   db.ArrServiceTypeSonarr,
	"prowlarr": db.ArrServiceTypeProwlarr,
	"seerr":    db.ArrServiceTypeSeerr,
}

func validateArrServiceRequest(req arrServiceRequest) string {
	if strings.TrimSpace(req.Name) == "" {
		return "name is required"
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if _, ok := validArrTypes[req.Type]; !ok {
		return "type must be one of: radarr, sonarr, seerr, prowlarr"
	}
	if strings.TrimSpace(req.URL) == "" {
		return "url is required"
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return "api_key is required"
	}
	return ""
}

// requireAdmin checks if the current user is an admin.
// Writes an error response and returns false if not admin or on error.
func (h *ArrServiceHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(userID)
	if err != nil {
		slog.Error("failed to check admin status", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return false
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "only admins can manage arr services")
		return false
	}
	return true
}

// HandleArrServices handles GET (list) and POST (create) on /api/arr-services
func (h *ArrServiceHandler) HandleArrServices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listArrServices(w, r)
	case http.MethodPost:
		h.createArrService(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleArrService handles GET, PUT, DELETE on /api/arr-services/{id}
func (h *ArrServiceHandler) HandleArrService(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/arr-services/{id} (tolerates trailing slash)
	id, ok := extractPathID(r.URL.Path, "/api/arr-services/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getArrService(w, r, id)
	case http.MethodPut:
		h.updateArrService(w, r, id)
	case http.MethodDelete:
		h.deleteArrService(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ArrServiceHandler) listArrServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.DB.ListArrServices()
	if err != nil {
		slog.Error("failed to list arr services", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to list services")
		return
	}

	dtos := make([]arrServiceDTO, 0, len(services))
	for i := range services {
		dtos = append(dtos, toArrServiceDTO(&services[i]))
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (h *ArrServiceHandler) createArrService(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	var req arrServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateArrServiceRequest(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	serviceType := validArrTypes[req.Type]
	service, err := h.DB.CreateArrService(strings.TrimSpace(req.Name), serviceType, strings.TrimSpace(req.URL), strings.TrimSpace(req.APIKey))
	if err != nil {
		slog.Error("failed to create arr service", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create service")
		return
	}

	writeJSON(w, http.StatusCreated, toArrServiceDTO(service))
}

func (h *ArrServiceHandler) getArrService(w http.ResponseWriter, r *http.Request, id string) {
	service, err := h.DB.GetArrService(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		slog.Error("failed to get arr service", slog.Any("service_id", id), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get service")
		return
	}

	writeJSON(w, http.StatusOK, toArrServiceDTO(service))
}

func (h *ArrServiceHandler) updateArrService(w http.ResponseWriter, r *http.Request, id string) {
	if !h.requireAdmin(w, r) {
		return
	}

	var req arrServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateArrServiceRequest(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	serviceType := validArrTypes[req.Type]
	service, err := h.DB.UpdateArrService(id, strings.TrimSpace(req.Name), serviceType, strings.TrimSpace(req.URL), strings.TrimSpace(req.APIKey))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		slog.Error("failed to update arr service", slog.Any("service_id", id), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to update service")
		return
	}

	writeJSON(w, http.StatusOK, toArrServiceDTO(service))
}

func (h *ArrServiceHandler) deleteArrService(w http.ResponseWriter, r *http.Request, id string) {
	if !h.requireAdmin(w, r) {
		return
	}

	if err := h.DB.DeleteArrService(id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		slog.Error("failed to delete arr service", slog.Any("service_id", id), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to delete service")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
