package handlers

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// captureRequest is the JSON request body for POST /api/books/capture.
type captureRequest struct {
	URL         string `json:"url"`
	LibraryID   string `json:"library_id"`
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
}

// captureAcceptedResponse is the JSON body returned on a successful capture request.
type captureAcceptedResponse struct {
	Message   string `json:"message"`
	URL       string `json:"url"`
	LibraryID string `json:"library_id"`
}

// HandleCapture handles POST /api/books/capture.
// It validates the requested URL, looks up the target library, and enqueues a
// capture:url background job that fetches the page, converts it to EPUB via
// go-readability + go-epub, and passes the resulting file through the existing
// process:file pipeline.
//
//	@Summary		Capture a web URL as an EPUB book
//	@Description	Fetch a web page, extract its article content, convert it to EPUB, and add it to a library. Processing happens asynchronously.
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		captureRequest			true	"URL capture request"
//	@Success		202		{object}	captureAcceptedResponse
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/books/capture [post]
func (h *BookHandler) HandleCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.Enqueuer == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "background processing not configured")
		return
	}

	var req captureRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "url is required")
		return
	}

	if !isValidCaptureURL(rawURL) {
		writeError(r.Context(), w, http.StatusBadRequest, "url must be a valid http or https URL")
		return
	}

	libraryID := strings.TrimSpace(req.LibraryID)
	if libraryID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "library_id is required")
		return
	}

	lib, err := h.DB.GetLibrary(r.Context(), libraryID)
	if handleDBErr(r.Context(), w, err, "library") {
		return
	}

	libraryRoot, err := parseFirstLibraryPath(lib.Paths)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to parse library paths",
			slog.String(otelkeys.LibraryID, libraryID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "invalid library configuration")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	payload := jobs.CaptureURLPayload{
		URL:                 rawURL,
		LibraryID:           libraryID,
		LibraryRoot:         libraryRoot,
		UserID:              userID,
		OverrideTitle:       strings.TrimSpace(req.Title),
		OverrideAuthor:      strings.TrimSpace(req.Author),
		OverrideDescription: strings.TrimSpace(req.Description),
		OverrideLanguage:    strings.TrimSpace(req.Language),
		OverridePublisher:   strings.TrimSpace(req.Publisher),
	}

	enqueueCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
	defer cancel()

	if _, err := h.Enqueuer.Enqueue(enqueueCtx, jobs.JobCaptureURL, payload); err != nil {
		slog.ErrorContext(r.Context(), "failed to enqueue capture:url job",
			slog.String(otelkeys.URL, rawURL),
			slog.String(otelkeys.LibraryID, libraryID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "failed to queue URL for capture")
		return
	}

	// Note: audit logging is handled by the background job (capture:url) after successful EPUB creation.
	// Logging here would produce duplicate audit entries.

	slog.InfoContext(r.Context(), "URL capture request accepted",
		slog.String(otelkeys.URL, rawURL),
		slog.String(otelkeys.LibraryID, libraryID),
	)

	writeJSON(r.Context(), w, http.StatusAccepted, captureAcceptedResponse{
		Message:   "URL accepted for capture",
		URL:       rawURL,
		LibraryID: libraryID,
	})
}

// dnsLookupTimeout is the maximum time allowed for a DNS lookup during URL
// validation.
const dnsLookupTimeout = 2 * time.Second

// isValidCaptureURL reports whether rawURL is a valid http or https URL and does
// not target private, reserved, or link-local IP addresses (SSRF protection).
// All resolved IPs must be public — if any resolved address is private, the URL
// is rejected to prevent DNS rebinding and mixed-record attacks.
func isValidCaptureURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Only allow http and https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Extract hostname; reject if empty or is a known private service name
	hostname := u.Hostname()
	if hostname == "" {
		return false
	}

	// Reject localhost-like hostnames
	if hostname == "localhost" || hostname == "localhost.localdomain" {
		return false
	}

	// Try to parse as IP address; if successful, validate it's not in a private range
	if ip := net.ParseIP(hostname); ip != nil {
		return isPublicIP(ip)
	}

	// For hostnames (not IPs), resolve them and check all resolved IPs are public
	lookupCtx, cancel := context.WithTimeout(context.Background(), dnsLookupTimeout)
	defer cancel()

	ipAddrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, hostname)
	if err != nil {
		// If resolution fails, reject the URL (can't verify it's safe)
		return false
	}

	if len(ipAddrs) == 0 {
		return false
	}

	// Reject if ANY resolved IP is non-public (prevents DNS rebinding / mixed records)
	for _, ipAddr := range ipAddrs {
		if !isPublicIP(ipAddr.IP) {
			return false
		}
	}

	return true
}

// isPublicIP reports whether ip is a globally routable unicast address.
func isPublicIP(ip net.IP) bool {
	return !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
