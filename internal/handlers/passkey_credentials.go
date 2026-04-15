package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandlePasskeyEnabled reports whether passkeys are configured on this server.
//
//	@Summary		Check if passkeys are enabled
//	@Description	Returns whether WebAuthn/passkey authentication is configured
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	map[string]bool
//	@Router			/auth/passkey/enabled [get]
func (h *PasskeyHandler) HandlePasskeyEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, map[string]bool{"enabled": h.WebAuthn != nil})
}

// HandlePasskeyCredentials dispatches GET (list) for /api/auth/passkey/credentials.
//
//	@Summary		List passkey credentials
//	@Description	List all passkey credentials for the authenticated user
//	@Tags			Auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		passkeyCredentialDTO
//	@Failure		401	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/auth/passkey/credentials [get]
func (h *PasskeyHandler) HandlePasskeyCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listUserEntities(w, r, "passkey credentials", h.DB.ListPasskeyCredentials, toPasskeyCredentialDTO)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandlePasskeyCredential dispatches DELETE for /api/auth/passkey/credentials/{id}.
//
//	@Summary		Delete a passkey credential
//	@Description	Delete a specific passkey credential owned by the authenticated user
//	@Tags			Auth
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Passkey credential ID"
//	@Success		204	"Passkey deleted"
//	@Failure		400	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		405	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/auth/passkey/credentials/{id} [delete]
func (h *PasskeyHandler) HandlePasskeyCredential(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/auth/passkey/credentials/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid passkey credential ID")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		deleteUserOwnedResource(h.DB, w, r, id, "passkey", "passkey", otelkeys.PasskeyCredentialID,
			h.DB.GetPasskeyCredential, h.DB.DeletePasskeyCredential,
			db.AuditActionPasskeyDeleted,
			func(c *db.PasskeyCredential) map[string]any { return map[string]any{"name": c.Name} },
		)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
