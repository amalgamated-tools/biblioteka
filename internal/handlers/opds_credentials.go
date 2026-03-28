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
