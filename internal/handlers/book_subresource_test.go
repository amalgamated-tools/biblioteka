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

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []fakeItemDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
		require.Len(t, dtos, 2)
		if dtos[0].Name != "Alpha" {
			t.Errorf("dtos[0].Name = %q, want %q", dtos[0].Name, "Alpha")
		}
		if dtos[1].Name != "Beta" {
			t.Errorf("dtos[1].Name = %q, want %q", dtos[1].Name, "Beta")
		}
	})

	t.Run("empty result returns empty JSON array", func(t *testing.T) {
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return []fakeItem{}, nil
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-1", getFn, toFakeItemDTO, "fake items")

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []fakeItemDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
		if len(dtos) != 0 {
			t.Errorf("len = %d, want 0", len(dtos))
		}
	})

	t.Run("getFn error returns 500", func(t *testing.T) {
		getFn := func(_ context.Context, _ string) ([]fakeItem, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-1", getFn, toFakeItemDTO, "fake items")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var resp errorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
		if resp.Error != "failed to get fake items" {
			t.Errorf("error = %q, want %q", resp.Error, "failed to get fake items")
		}
	})

	t.Run("bookID is forwarded to getFn", func(t *testing.T) {
		var capturedID string
		getFn := func(_ context.Context, id string) ([]fakeItem, error) {
			capturedID = id
			return []fakeItem{}, nil
		}

		w := httptest.NewRecorder()
		respondBookSubResource(t.Context(), w, "book-xyz", getFn, toFakeItemDTO, "fake items")

		if capturedID != "book-xyz" {
			t.Errorf("capturedID = %q, want %q", capturedID, "book-xyz")
		}
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

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
		if len(capturedPayload) != 1 || capturedPayload[0] != "id-1" {
			t.Errorf("capturedPayload = %v, want [id-1]", capturedPayload)
		}
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

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
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

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var resp errorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
		if resp.Error != "failed to set fake items" {
			t.Errorf("error = %q, want %q", resp.Error, "failed to set fake items")
		}
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

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}
