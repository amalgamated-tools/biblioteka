package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// OPDSCredentialHandler manages OPDS credentials via the JSON API.
type OPDSCredentialHandler struct {
	DB *db.DB
}

// HandleOPDSCredentials godoc
//
//	@Summary		Manage OPDS credentials
//	@Description	GET returns the current user's OPDS credential.
//	@Description	PUT creates or replaces the current user's OPDS credential using a JSON body of type credentialRequest (username and password, with the password hashed server-side).
//	@Description	DELETE removes the current user's OPDS credential.
//	@Tags			opds-credentials
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	credentialResponse	"Credential returned or updated"
//	@Success		204	"Credential deleted"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"No OPDS credentials configured"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		409	{object}	errorResponse	"Username already taken"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/opds/credentials [get]
//	@Router			/opds/credentials [put]
//	@Router			/opds/credentials [delete]
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
		getByUserID:     credentialGetAdapter(h.DB.GetOPDSCredentialByUserID),
		upsert:          credentialUpsertAdapter(h.DB.UpsertOPDSCredential),
		del:             h.DB.DeleteOPDSCredential,
	}
}
