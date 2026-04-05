package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeItem is a simple entity used by book_subresource tests.
type fakeItem struct {
	ID   string
	Name string
}

// fakeItemDTO is the DTO form of fakeItem.
type fakeItemDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toFakeItemDTO(f *fakeItem) fakeItemDTO {
	return fakeItemDTO{ID: f.ID, Name: f.Name}
}

func TestRespondBookSubResource(t *testing.T) {
	t.Run("success returns 200 with JSON array", func(t *testing.T) {
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return []fakeItem{{ID: "1", Name: "Alpha"}, {ID: "2", Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-1", getFn, toFakeItemDTO, "fake items")

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []fakeItemDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
		require.Len(t, dtos, 2)
		require.Equal(t, "Alpha", dtos[0].Name)
		require.Equal(t, "Beta", dtos[1].Name)
	})

	t.Run("empty result returns empty JSON array", func(t *testing.T) {
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return []fakeItem{}, nil
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-1", getFn, toFakeItemDTO, "fake items")

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []fakeItemDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
		require.Len(t, dtos, 0)
	})

	t.Run("getFn error returns 500", func(t *testing.T) {
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-1", getFn, toFakeItemDTO, "fake items")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var resp errorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
		require.Equal(t, "failed to get fake items", resp.Error)
	})

	t.Run("bookID is forwarded to getFn", func(t *testing.T) {
		var capturedID string
		getFn := func(_ context.Context, id string) ([]fakeItem, error) {
			capturedID = id
			return []fakeItem{}, nil
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-xyz", getFn, toFakeItemDTO, "fake items")

		require.Equal(t, "book-xyz", capturedID)
	})
}

// fakeSetRequest is the request body for putBookSubResource tests.
type fakeSetRequest struct {
	IDs []string `json:"ids"`
}

func TestPutBookSubResource(t *testing.T) {
	t.Run("success decodes and sets then re-fetches", func(t *testing.T) {
		var capturedPayload []string
		setFn := func(_ context.Context, _ string, ids []string) error {
			capturedPayload = ids
			return nil
		}
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return []fakeItem{{ID: capturedPayload[0], Name: "Alpha"}}, nil
		}

		body := mustMarshal(t, fakeSetRequest{IDs: []string{"id-1"}})
		r := httptest.NewRequest(http.MethodPut, "/api/books/book-1/fake", bytes.NewReader(body))
		w := httptest.NewRecorder()

		putBookSubResource(w, r, "book-1",
			getFn, setFn,
			func(req *fakeSetRequest) []string { return req.IDs },
			toFakeItemDTO,
			"fake items",
		)

		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, capturedPayload, 1)
		require.Equal(t, "id-1", capturedPayload[0])
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		setFn := func(_ context.Context, _ string, _ []string) error { return nil }
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) { return nil, nil }

		r := httptest.NewRequest(http.MethodPut, "/api/books/book-1/fake", strings.NewReader("not-json"))
		w := httptest.NewRecorder()

		putBookSubResource(w, r, "book-1",
			getFn, setFn,
			func(req *fakeSetRequest) []string { return req.IDs },
			toFakeItemDTO,
			"fake items",
		)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("setFn error returns 500", func(t *testing.T) {
		setFn := func(_ context.Context, _ string, _ []string) error {
			return errors.New("set failed")
		}
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) { return nil, nil }

		body := mustMarshal(t, fakeSetRequest{IDs: []string{"id-1"}})
		r := httptest.NewRequest(http.MethodPut, "/api/books/book-1/fake", bytes.NewReader(body))
		w := httptest.NewRecorder()

		putBookSubResource(w, r, "book-1",
			getFn, setFn,
			func(req *fakeSetRequest) []string { return req.IDs },
			toFakeItemDTO,
			"fake items",
		)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var resp errorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
		require.Equal(t, "failed to set fake items", resp.Error)
	})

	t.Run("getFn error after set returns 500", func(t *testing.T) {
		setFn := func(_ context.Context, _ string, _ []string) error { return nil }
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return nil, errors.New("get failed after set")
		}

		body := mustMarshal(t, fakeSetRequest{IDs: []string{"id-1"}})
		r := httptest.NewRequest(http.MethodPut, "/api/books/book-1/fake", bytes.NewReader(body))
		w := httptest.NewRecorder()

		putBookSubResource(w, r, "book-1",
			getFn, setFn,
			func(req *fakeSetRequest) []string { return req.IDs },
			toFakeItemDTO,
			"fake items",
		)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
