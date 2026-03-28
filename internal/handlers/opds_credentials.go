package handlers

import (
	"context"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// OPDSCredentialHandler manages OPDS credentials via the JSON API.
type OPDSCredentialHandler struct {
	DB *db.DB
}

// GetOPDSCredentials handles GET /api/opds/credentials.
//
//	@Summary		Get OPDS credentials
//	@Description	GET returns the current user's OPDS credential.
//	@Tags			opds-credentials
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	credentialResponse	"Credential returned"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"No OPDS credentials configured"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/opds/credentials [get]
func (h *OPDSCredentialHandler) GetOPDSCredentials(w http.ResponseWriter, r *http.Request) {
	handleCredentials(h.credOps(), w, r)
}

// PutOPDSCredentials handles PUT /api/opds/credentials.
//
//	@Summary		Create or update OPDS credentials
//	@Description	PUT creates or replaces the current user's OPDS credential (username and password, with the password hashed server-side).
//	@Tags			opds-credentials
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		credentialRequest	false	"Credential data (required for PUT)"
//	@Success		200	{object}	credentialResponse	"Credential created or updated"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"No OPDS credentials configured"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		409	{object}	errorResponse	"Username already taken"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/opds/credentials [put]
func (h *OPDSCredentialHandler) PutOPDSCredentials(w http.ResponseWriter, r *http.Request) {
	handleCredentials(h.credOps(), w, r)
}

// DeleteOPDSCredentials handles DELETE /api/opds/credentials.
//
//	@Summary		Delete OPDS credentials
//	@Description	DELETE removes the current user's OPDS credential.
//	@Tags			opds-credentials
//	@Security		BearerAuth
//	@Produce		json
//	@Success		204	"Credential deleted"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"No OPDS credentials configured"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/opds/credentials [delete]
func (h *OPDSCredentialHandler) DeleteOPDSCredentials(w http.ResponseWriter, r *http.Request) {
	handleCredentials(h.credOps(), w, r)
}

// HandleOPDSCredentials dispatches GET/PUT/DELETE for /api/opds/credentials.
func (h *OPDSCredentialHandler) HandleOPDSCredentials(w http.ResponseWriter, r *http.Request) {
	handleCredentials(h.credOps(), w, r)
}

func (h *OPDSCredentialHandler) credOps() credentialOps {
	return credentialOps{
		db:              h.DB,
		protocol:        "OPDS",
		auditEntityType: "opds_credential",
		auditUpsert:     db.AuditActionOPDSCredentialUpdated,
		auditDelete:     db.AuditActionOPDSCredentialDeleted,
		errConflict:     db.ErrOPDSUsernameExists,
		getByUserID: func(ctx context.Context, userID string) (credentialEntity, error) {
			c, err := h.DB.GetOPDSCredentialByUserID(ctx, userID)
			if err != nil {
				return credentialEntity{}, err
			}
			return credentialEntity{ID: c.ID, Username: c.Username, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}, nil
		},
		upsert: func(ctx context.Context, userID, username, hash string) (credentialEntity, error) {
			c, err := h.DB.UpsertOPDSCredential(ctx, userID, username, hash)
			if err != nil {
				return credentialEntity{}, err
			}
			return credentialEntity{ID: c.ID, Username: c.Username, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}, nil
		},
		del: h.DB.DeleteOPDSCredential,
	}
}
