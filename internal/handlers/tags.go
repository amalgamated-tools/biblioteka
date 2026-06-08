package handlers

import (
	"context"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// TagHandler holds dependencies for tag endpoints.
type TagHandler struct {
	DB *db.DB
}

type tagRequest struct {
	Name string `json:"name"`
}

type tagDTO struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

func toTagDTO(t *db.Tag) tagDTO {
	return tagDTO{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// tagOps returns the namedEntityOps configuration for the Tag entity.
func (h *TagHandler) tagOps() namedEntityOps[db.Tag, tagDTO, tagRequest] {
	return namedEntityOps[db.Tag, tagDTO, tagRequest]{
		db:             h.DB,
		entityLabel:    "tag",
		entityArticle:  "a tag",
		idKey:          otelkeys.TagID,
		errInvalidName: db.ErrInvalidTagName,
		errNameExists:  db.ErrTagNameExists,
		auditCreate:    db.AuditActionTagCreated,
		auditUpdate:    db.AuditActionTagUpdated,
		get:            h.DB.GetTag,
		create: func(ctx context.Context, req tagRequest) (*db.Tag, error) {
			return h.DB.CreateTag(ctx, req.Name)
		},
		update: func(ctx context.Context, id string, req tagRequest) (*db.Tag, error) {
			return h.DB.UpdateTag(ctx, id, req.Name)
		},
		reqName:    func(req tagRequest) string { return req.Name },
		entityName: func(t *db.Tag) string { return t.Name },
		entityID:   func(t *db.Tag) string { return t.ID },
		toDTO:      toTagDTO,
	}
}

// HandleTags handles GET /api/tags and POST /api/tags.
//
//	@Summary		List or create tags
//	@Description	GET returns paginated tags. POST creates a new tag.
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int	false	"Number of items to skip (default 0)"
//	@Failure		401	{object}	errorResponse
//	@Success		200	{object}	tagListDTO
//	@Success		201	{object}	tagDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		409	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/tags [get]
//	@Router			/tags [post]
func (h *TagHandler) HandleTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTags(w, r)
	case http.MethodPost:
		h.createTag(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleTag handles requests under /api/tags/{id}.
//
//	@Summary		Get, update, or delete a tag
//	@Description	GET returns a tag by ID. PUT updates a tag. DELETE removes a tag.
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Tag ID"
//	@Failure		401	{object}	errorResponse
//	@Success		200	{object}	tagDTO
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		409	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/tags/{id} [get]
//	@Router			/tags/{id} [put]
//	@Router			/tags/{id} [delete]
func (h *TagHandler) HandleTag(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/tags/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTag(w, r, id)
	case http.MethodPut:
		h.updateTag(w, r, id)
	case http.MethodDelete:
		h.deleteTag(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type tagListDTO struct {
	Tags []tagDTO `json:"tags"`
	paginationMeta
}

func (h *TagHandler) listTags(w http.ResponseWriter, r *http.Request) {
	listPaginatedEntities(w, r, "tags", h.DB.ListTagsPaginated, toTagDTO,
		func(items []tagDTO, total, limit, offset int) tagListDTO {
			return tagListDTO{
				Tags: items,
				paginationMeta: paginationMeta{
					Total:  total,
					Limit:  limit,
					Offset: offset,
				},
			}
		},
	)
}

func (h *TagHandler) createTag(w http.ResponseWriter, r *http.Request) {
	createNamedEntity(h.tagOps(), w, r)
}

func (h *TagHandler) getTag(w http.ResponseWriter, r *http.Request, id string) {
	getNamedEntity(h.tagOps(), w, r, id)
}

func (h *TagHandler) updateTag(w http.ResponseWriter, r *http.Request, id string) {
	updateNamedEntity(h.tagOps(), w, r, id)
}

func (h *TagHandler) deleteTag(w http.ResponseWriter, r *http.Request, id string) {
	deleteResource(h.DB, w, r, id, "tag", "tag", otelkeys.TagID,
		h.DB.GetTag, h.DB.DeleteTag,
		db.AuditActionTagDeleted,
		func(t *db.Tag) map[string]any { return map[string]any{"name": t.Name} },
	)
}
