package handlers

import (
	"crypto/md5" // #nosec G501 -- MD5 is required by the KOReader kosync protocol
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// KOSyncHandler implements the KOReader kosync-compatible progress sync API
// and the Biblioteka credential-management API for KOSync.
type KOSyncHandler struct {
	DB *db.DB
}

// kosyncProgressResponse is the wire format returned by the progress endpoints.
// The timestamp field is a Unix epoch second, matching the KOReader kosync protocol.
type kosyncProgressResponse struct {
	Document   string  `json:"document"`
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device,omitempty"`
	DeviceID   string  `json:"device_id,omitempty"`
	Timestamp  int64   `json:"timestamp"`
}

// kosyncProgressRequest is the body for PUT /api/syncs/progress.
type kosyncProgressRequest struct {
	Document   string  `json:"document"`
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device"`
	DeviceID   string  `json:"device_id"`
}

// ---- Biblioteka credential-management API (JWT-protected) ----

// HandleKOSyncCredentials godoc
//
//	@Summary		Manage KOSync credentials
//	@Description	GET returns the current user's KOSync credential (username and timestamps; the hashed password is never returned).
//	@Description	PUT creates or updates the current user's KOSync credential using a JSON body matching credentialRequest.
//	@Description	DELETE removes the current user's KOSync credential.
//	@Tags			kosync-credentials
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	credentialResponse	"Credential returned or updated"
//	@Success		204	"Credential deleted"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"No KOSync credentials configured"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		409	{object}	errorResponse	"Username already taken"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/kosync/credentials [get]
//	@Router			/kosync/credentials [put]
//	@Router			/kosync/credentials [delete]
func (h *KOSyncHandler) HandleKOSyncCredentials(w http.ResponseWriter, r *http.Request) {
	handleCredentials(h.credOps(), w, r)
}

func (h *KOSyncHandler) credOps() credentialOps {
	return credentialOps{
		db:              h.DB,
		protocol:        "KOSync",
		auditEntityType: "kosync_credential",
		auditUpsert:     db.AuditActionKOSyncCredentialUpdated,
		auditDelete:     db.AuditActionKOSyncCredentialDeleted,
		errConflict:     db.ErrKOSyncUsernameExists,
		getByUserID:     credGetAdapter(h.DB.GetKOSyncCredentialByUserID),
		upsert:          credUpsertAdapter(h.DB.UpsertKOSyncCredential),
		del:             h.DB.DeleteKOSyncCredential,
		deriveKey:       kosyncProtocolKey,
	}
}

// ---- KOReader kosync-compatible protocol endpoints ----

// HandleKOSyncUserCreate handles POST /api/user/create.
// KOReader always attempts to register before authenticating.  Because account
// creation on this server is managed through the Biblioteka web UI (not through
// the kosync protocol), this endpoint always returns HTTP 409 Conflict so that
// KOReader falls through to the authentication step.
func (h *KOSyncHandler) HandleKOSyncUserCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// 409 tells KOReader "username already exists" and causes it to proceed to
	// the /api/user/auth step.  Users must set up KOSync credentials via the
	// Biblioteka web interface before connecting KOReader.
	writeError(r.Context(), w, http.StatusConflict, "account creation is managed through the Biblioteka web interface")
}

// HandleKOSyncUserAuth handles GET /api/user/auth.
// This endpoint is called by KOReader to verify credentials after the create
// step.  Authentication is performed by the requireKOSyncAuth middleware; if
// this handler is reached the credentials are already valid.
func (h *KOSyncHandler) HandleKOSyncUserAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"authorized": "OK"})
}

// HandleKOSyncProgress handles PUT and GET on /api/syncs/progress and
// GET on /api/syncs/progress/{document}.
func (h *KOSyncHandler) HandleKOSyncProgress(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		h.putProgress(w, r)
	case http.MethodGet:
		h.getProgress(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Maximum lengths for KOSync progress fields to prevent abuse.
const (
	maxDocumentLen = 1024
	maxProgressLen = 4096
	maxDeviceLen   = 256
)

func (h *KOSyncHandler) putProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	var req kosyncProgressRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if req.Document == "" {
		writeError(ctx, w, http.StatusBadRequest, "document is required")
		return
	}
	if strings.Contains(req.Document, "/") {
		writeError(ctx, w, http.StatusBadRequest, "document identifier must not contain '/'")
		return
	}
	if len(req.Document) > maxDocumentLen {
		writeError(ctx, w, http.StatusBadRequest, "document identifier too long")
		return
	}
	if req.Progress == "" {
		writeError(ctx, w, http.StatusBadRequest, "progress is required")
		return
	}
	if len(req.Progress) > maxProgressLen {
		writeError(ctx, w, http.StatusBadRequest, "progress value too long")
		return
	}
	if req.Percentage < 0 || req.Percentage > 1 || math.IsNaN(req.Percentage) || math.IsInf(req.Percentage, 0) {
		writeError(ctx, w, http.StatusBadRequest, "percentage must be between 0 and 1")
		return
	}
	if len(req.Device) > maxDeviceLen {
		writeError(ctx, w, http.StatusBadRequest, "device name too long")
		return
	}
	if len(req.DeviceID) > maxDeviceLen {
		writeError(ctx, w, http.StatusBadRequest, "device ID too long")
		return
	}

	var device, deviceID *string
	if req.Device != "" {
		device = &req.Device
	}
	if req.DeviceID != "" {
		deviceID = &req.DeviceID
	}

	p, err := h.DB.UpsertReadingProgress(ctx, userID, req.Document, req.Progress, req.Percentage, device, deviceID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert reading progress",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Document, req.Document),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to update progress")
		return
	}

	writeJSON(ctx, w, http.StatusOK, toKOSyncProgressResponse(p))
}

func (h *KOSyncHandler) getProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	const prefix = "/api/syncs/progress/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeError(ctx, w, http.StatusBadRequest, "document identifier is required")
		return
	}

	document := r.URL.Path[len(prefix):]
	if document == "" {
		writeError(ctx, w, http.StatusBadRequest, "document identifier is required")
		return
	}
	if len(document) > maxDocumentLen {
		writeError(ctx, w, http.StatusBadRequest, "document identifier too long")
		return
	}

	p, err := h.DB.GetReadingProgress(ctx, userID, document)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "no progress found for document")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get reading progress",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Document, document),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get progress")
		return
	}

	writeJSON(ctx, w, http.StatusOK, toKOSyncProgressResponse(p))
}

func toKOSyncProgressResponse(p *db.ReadingProgress) kosyncProgressResponse {
	resp := kosyncProgressResponse{
		Document:   p.Document,
		Progress:   p.Progress,
		Percentage: p.Percentage,
		Timestamp:  p.UpdatedAt.Unix(),
	}
	if p.Device != nil {
		resp.Device = *p.Device
	}
	if p.DeviceID != nil {
		resp.DeviceID = *p.DeviceID
	}
	return resp
}

// kosyncProtocolKey converts a plain-text credential into the hex-encoded MD5
// digest that KOReader transmits as x-auth-key.  MD5 is mandated by the kosync
// wire protocol; the returned value is always bcrypt-hashed before storage, so
// MD5 is never the final layer of protection.
func kosyncProtocolKey(plaintext string) string {
	digest := md5.Sum([]byte(plaintext)) // #nosec G401 -- kosync protocol requirement
	return hex.EncodeToString(digest[:])
}
