package handlers

import (
	"context"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// groupOps returns the namedEntityOps configuration for the
// ReadingGroup entity.
func (h *GroupHandler) groupOps() namedEntityOps[db.ReadingGroup, groupDTO, groupRequest] {
	return namedEntityOps[db.ReadingGroup, groupDTO, groupRequest]{
		db:              h.DB,
		entityLabel:     "group",
		entityArticle:   "a group",
		idKey:           otelkeys.GroupID,
		auditEntityType: "group",
		errInvalidName:  db.ErrInvalidGroupName,
		errNameExists:   db.ErrGroupNameExists,
		auditCreate:     db.AuditActionGroupCreated,
		auditUpdate:     db.AuditActionGroupUpdated,
		// These closures enforce user isolation by extracting the authenticated
		// user ID from the request context before delegating to the user-scoped
		// DB methods.
		get: func(ctx context.Context, id string) (*db.ReadingGroup, error) {
			return h.DB.GetGroup(ctx, id, auth.UserIDFromContext(ctx))
		},
		create: func(ctx context.Context, req groupRequest) (*db.ReadingGroup, error) {
			return h.DB.CreateGroup(ctx, auth.UserIDFromContext(ctx), req.Name, req.Description)
		},
		update: func(ctx context.Context, id string, req groupRequest) (*db.ReadingGroup, error) {
			return h.DB.UpdateGroup(ctx, id, auth.UserIDFromContext(ctx), req.Name, req.Description)
		},
		reqName:    func(req groupRequest) string { return req.Name },
		entityName: func(g *db.ReadingGroup) string { return g.Name },
		entityID:   func(g *db.ReadingGroup) string { return g.ID },
		toDTO:      toGroupDTO,
	}
}

// listGroups returns all groups for the authenticated user.
//
//	@Summary		List groups
//	@Description	Returns all reading groups the authenticated user belongs to
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		groupDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/groups [get]
func (h *GroupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	listUserEntities(w, r, "groups", h.DB.ListGroups, toGroupDTO)
}

// createGroup creates a new reading group.
//
//	@Summary		Create a group
//	@Description	Create a new reading group owned by the authenticated user
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		groupRequest	true	"Group data"
//	@Success		201		{object}	groupDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups [post]
func (h *GroupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	createNamedEntity(h.groupOps(), w, r)
}

func (h *GroupHandler) handleGroup(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r, id)
	case http.MethodPut:
		h.updateGroup(w, r, id)
	case http.MethodDelete:
		h.deleteGroup(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getGroup returns a single group by ID.
//
//	@Summary		Get a group
//	@Description	Returns a single reading group by ID (requester must be a member)
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{object}	groupDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id} [get]
func (h *GroupHandler) getGroup(w http.ResponseWriter, r *http.Request, id string) {
	getNamedEntity(h.groupOps(), w, r, id)
}

// updateGroup updates an existing group.
//
//	@Summary		Update a group
//	@Description	Update the name and description of a reading group (owner only)
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"Group ID"
//	@Param			body	body		groupRequest	true	"Group data"
//	@Success		200		{object}	groupDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id} [put]
func (h *GroupHandler) updateGroup(w http.ResponseWriter, r *http.Request, id string) {
	updateNamedEntity(h.groupOps(), w, r, id)
}

// deleteGroup deletes a reading group.
//
//	@Summary		Delete a group
//	@Description	Delete a reading group (owner only)
//	@Tags			Groups
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id} [delete]
func (h *GroupHandler) deleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	deleteUserOwnedResource(h.DB, w, r, id, "group", "group", otelkeys.GroupID,
		h.DB.GetGroup, h.DB.DeleteGroup,
		db.AuditActionGroupDeleted,
		func(g *db.ReadingGroup) map[string]any { return map[string]any{"name": g.Name} },
	)
}
