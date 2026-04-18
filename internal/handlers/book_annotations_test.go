package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// setupAnnotationHandler creates a BookAnnotationHandler wired into a BookHandler,
// a test user, and a test book. Returns the BookHandler (for sub-resource route
// dispatch), the standalone BookAnnotationHandler (for /api/annotations/{id}
// tests), the user ID, and the book ID.
func setupAnnotationHandler(t *testing.T) (*BookHandler, *BookAnnotationHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	ah := &BookAnnotationHandler{DB: d}
	h := &BookHandler{DB: d, AnnotationHandler: ah}
	user, err := d.CreateUser(t.Context(), "Anno User", "anno@example.com", "password1")
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Annotated Book"})
	require.NoError(t, err)
	return h, ah, user.ID, book.ID
}

// ---- GET /api/books/{id}/annotations ----

func TestListBookAnnotations_Empty(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+bookID+"/annotations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 0)
}

func TestListBookAnnotations_WithAnnotations(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateAnnotation(ctx, userID, bookID, "Note one", nil, nil)
	require.NoError(t, err)
	_, err = h.DB.CreateAnnotation(ctx, userID, bookID, "Note two", nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+bookID+"/annotations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2)
}

func TestListBookAnnotations_IsolatedByUser(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	other, err := h.DB.CreateUser(ctx, "Other", "other@example.com", "pw2")
	require.NoError(t, err)
	_, err = h.DB.CreateAnnotation(ctx, other.ID, bookID, "Other's note", nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+bookID+"/annotations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 0)
}

func TestListBookAnnotations_MethodNotAllowed(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+bookID+"/annotations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---- POST /api/books/{id}/annotations ----

func TestCreateBookAnnotation_Success(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)

	cfi := "/p[1]/s[2]"
	body := mustMarshal(t, createAnnotationRequest{Text: "Great passage", CFI: &cfi})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/annotations", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var dto annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.NotEmpty(t, dto.ID)
	require.Equal(t, "Great passage", dto.Text)
	require.NotNil(t, dto.CFI)
	require.Equal(t, cfi, *dto.CFI)
	require.Equal(t, userID, dto.UserID)
	require.Equal(t, bookID, dto.BookID)
}

func TestCreateBookAnnotation_InvalidJSON(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/annotations", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateBookAnnotation_NonMemberGroupForbidden(t *testing.T) {
	h, _, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	owner, err := h.DB.CreateUser(ctx, "Owner", "owner@example.com", "pw")
	require.NoError(t, err)
	g, err := h.DB.CreateGroup(ctx, owner.ID, "Book Club", nil)
	require.NoError(t, err)

	body := mustMarshal(t, createAnnotationRequest{Text: "Sneaky note", GroupID: &g.ID})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/annotations", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

// ---- GET /api/annotations/{id} ----

func TestGetAnnotation_Success(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "My note", nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/annotations/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, a.ID, dto.ID)
	require.Equal(t, "My note", dto.Text)
}

func TestGetAnnotation_NotFound(t *testing.T) {
	_, ah, userID, _ := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/annotations/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnnotation_OtherUserCannotAccess(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Private note", nil, nil)
	require.NoError(t, err)

	other, err := ah.DB.CreateUser(ctx, "Other", "other2@example.com", "pw")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/annotations/"+a.ID, nil)
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnnotation_InvalidID(t *testing.T) {
	_, ah, userID, _ := setupAnnotationHandler(t)

	// Path without a valid ID segment (empty after prefix)
	r := httptest.NewRequest(http.MethodGet, "/api/annotations/", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnnotation_MethodNotAllowed(t *testing.T) {
	_, ah, userID, _ := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/annotations/some-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---- PUT /api/annotations/{id} ----

func TestUpdateAnnotation_Success(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Original text", nil, nil)
	require.NoError(t, err)

	body := mustMarshal(t, updateAnnotationRequest{Text: "Updated text"})
	r := httptest.NewRequest(http.MethodPut, "/api/annotations/"+a.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto annotationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "Updated text", dto.Text)
}

func TestUpdateAnnotation_NotFound(t *testing.T) {
	_, ah, userID, _ := setupAnnotationHandler(t)

	body := mustMarshal(t, updateAnnotationRequest{Text: "Updated"})
	r := httptest.NewRequest(http.MethodPut, "/api/annotations/nonexistent", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAnnotation_OtherUserCannotUpdate(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Original", nil, nil)
	require.NoError(t, err)

	other, err := ah.DB.CreateUser(ctx, "Other", "other3@example.com", "pw")
	require.NoError(t, err)

	body := mustMarshal(t, updateAnnotationRequest{Text: "Hijacked"})
	r := httptest.NewRequest(http.MethodPut, "/api/annotations/"+a.ID, bytes.NewReader(body))
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAnnotation_NonMemberGroupForbidden(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Original", nil, nil)
	require.NoError(t, err)

	owner, err := ah.DB.CreateUser(ctx, "Owner", "owner2@example.com", "pw")
	require.NoError(t, err)
	g, err := ah.DB.CreateGroup(ctx, owner.ID, "Club", nil)
	require.NoError(t, err)

	body := mustMarshal(t, updateAnnotationRequest{Text: "Shared", GroupID: &g.ID})
	r := httptest.NewRequest(http.MethodPut, "/api/annotations/"+a.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateAnnotation_InvalidJSON(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Original", nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPut, "/api/annotations/"+a.ID, strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- DELETE /api/annotations/{id} ----

func TestDeleteAnnotation_Success(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "To be deleted", nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/annotations/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's gone.
	annotations, err := ah.DB.ListAnnotationsForBook(ctx, bookID, userID)
	require.NoError(t, err)
	require.Empty(t, annotations)
}

func TestDeleteAnnotation_NotFound(t *testing.T) {
	_, ah, userID, _ := setupAnnotationHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/annotations/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAnnotation_OtherUserCannotDelete(t *testing.T) {
	_, ah, userID, bookID := setupAnnotationHandler(t)
	ctx := t.Context()

	a, err := ah.DB.CreateAnnotation(ctx, userID, bookID, "Mine", nil, nil)
	require.NoError(t, err)

	other, err := ah.DB.CreateUser(ctx, "Other", "other4@example.com", "pw")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/annotations/"+a.ID, nil)
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()

	ah.HandleAnnotation(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}
