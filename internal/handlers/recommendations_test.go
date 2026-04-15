package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupRecommendationHandler(t *testing.T) (*RecommendationHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &RecommendationHandler{DB: d}
	u, err := d.CreateUser(t.Context(), "Rec User", "rec@example.com", "secret")
	require.NoError(t, err, "create user")
	return h, u.ID
}

func TestHandleRecommendations_EmptyLibrary(t *testing.T) {
	h, userID := setupRecommendationHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/recommendations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRecommendations(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []bookSummaryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	// Empty library → empty array (not null).
	require.NotNil(t, dtos)
	require.Empty(t, dtos)
}

func TestHandleRecommendations_ReturnsBooks(t *testing.T) {
	h, userID := setupRecommendationHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/recommendations", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRecommendations(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []bookSummaryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 1)
	require.Equal(t, "Dune", dtos[0].Title)
}

func TestHandleRecommendations_LimitParam(t *testing.T) {
	h, userID := setupRecommendationHandler(t)

	for i := range 5 {
		_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Book " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/recommendations?limit=3", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRecommendations(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []bookSummaryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.LessOrEqual(t, len(dtos), 3, "limit=3 should return at most 3 books")
}

func TestHandleRecommendations_LimitClamped(t *testing.T) {
	h, userID := setupRecommendationHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/recommendations?limit=9999", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRecommendations(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRecommendations_MethodNotAllowed(t *testing.T) {
	h, userID := setupRecommendationHandler(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		r := httptest.NewRequest(method, "/api/recommendations", nil)
		r = withUserID(r, userID)
		w := httptest.NewRecorder()

		h.HandleRecommendations(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code, "method %s should return 405", method)
	}
}
