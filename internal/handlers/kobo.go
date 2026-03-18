package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// KoboHandler handles Kobo sync device API endpoints and Kobo token management.
type KoboHandler struct {
	DB  *db.DB
	mux *http.ServeMux
}

// RegisterRoutes builds the internal sub-mux for Kobo device API routes.
// The middleware strips /kobo/{token} from the path before dispatching here.
func (h *KoboHandler) RegisterRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/v1/initialization", h.HandleInit)
	h.mux.HandleFunc("/v1/auth/device", h.HandleAuth)
	h.mux.HandleFunc("/v1/auth/refresh", h.HandleAuth)
	h.mux.HandleFunc("/v1/auth/exchange", h.HandleAuth)
	h.mux.HandleFunc("/v1/library/sync", h.HandleSync)
	h.mux.HandleFunc("/v1/library/", h.handleLibraryRoute)
	h.mux.HandleFunc("/download/", h.HandleDownload)
	h.mux.HandleFunc("/covers/", h.HandleCoverImage)
	h.mux.HandleFunc("/v1/user/loyalty/benefits", h.handleLoyaltyBenefits)
	h.mux.HandleFunc("/v1/analytics/gettests", h.handleAnalyticsGetTests)
	h.mux.HandleFunc("/", h.handleDefault)
}

// ServeHTTP dispatches Kobo device API requests through the sub-mux.
func (h *KoboHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleLibraryRoute dispatches /v1/library/{uuid}/metadata and /v1/library/{uuid}/state.
func (h *KoboHandler) handleLibraryRoute(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/metadata"):
		h.HandleBookMetadata(w, r)
	case strings.HasSuffix(r.URL.Path, "/state"):
		h.HandleBookState(w, r)
	default:
		writeKoboJSON(w, http.StatusOK, map[string]any{})
	}
}

func (h *KoboHandler) handleLoyaltyBenefits(w http.ResponseWriter, _ *http.Request) {
	writeKoboJSON(w, http.StatusOK, map[string]any{"Benefits": map[string]any{}})
}

func (h *KoboHandler) handleAnalyticsGetTests(w http.ResponseWriter, r *http.Request) {
	userKey := r.Header.Get("X-Kobo-userkey")
	writeKoboJSON(w, http.StatusOK, map[string]any{
		"Result":  "Success",
		"TestKey": userKey,
		"Tests":   map[string]any{},
	})
}

func (h *KoboHandler) handleDefault(w http.ResponseWriter, _ *http.Request) {
	writeKoboJSON(w, http.StatusOK, map[string]any{})
}

// writeKoboJSON writes a JSON response with the content type expected by Kobo devices.
func writeKoboJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// schemeAndHost returns the scheme and host for building absolute URLs.
func schemeAndHost(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		normalized := strings.ToLower(strings.TrimSpace(proto))
		if normalized == "http" || normalized == "https" {
			scheme = normalized
		}
	}
	host := r.Host
	return scheme + "://" + host
}
