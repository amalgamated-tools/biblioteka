package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ListEntities(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string `json:"label"`
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("error yields 500", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
		if result["error"] != "failed to list widgets" {
			t.Errorf("error = %q, want %q", result["error"], "failed to list widgets")
		}
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		if len(dtos) != 2 {
			t.Fatalf("len = %d, want 2", len(dtos))
		}
		if dtos[0].Label != "Alpha" {
			t.Errorf("dtos[0].Label = %q, want %q", dtos[0].Label, "Alpha")
		}
		if dtos[1].Label != "Beta" {
			t.Errorf("dtos[1].Label = %q, want %q", dtos[1].Label, "Beta")
		}
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		if len(dtos) != 0 {
			t.Errorf("len = %d, want 0", len(dtos))
		}
	})
}

func Test_MapSlice(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("converts elements", func(t *testing.T) {
		items := []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
		result := mapSlice(items, toDTO)
		if len(result) != 2 {
			t.Fatalf("len = %d, want 2", len(result))
		}
		if result[0].Label != "Alpha" {
			t.Errorf("result[0].Label = %q, want %q", result[0].Label, "Alpha")
		}
		if result[1].Label != "Beta" {
			t.Errorf("result[1].Label = %q, want %q", result[1].Label, "Beta")
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := mapSlice([]entity{}, toDTO)
		if result == nil {
			t.Fatal("result is nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("nil input returns empty slice", func(t *testing.T) {
		result := mapSlice(nil, toDTO)
		if result == nil {
			t.Fatal("result is nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})
}

func Test_ListUserEntities(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string `json:"label"`
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("error yields 500", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
		if result["error"] != "failed to list tokens" {
			t.Errorf("error = %q, want %q", result["error"], "failed to list tokens")
		}
	})

	t.Run("passes user ID to list function", func(t *testing.T) {
		var capturedUserID string
		listFn := func(_ context.Context, userID string) ([]entity, error) {
			capturedUserID = userID
			return []entity{{ID: 1, Name: "Alpha"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-42")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if capturedUserID != "user-42" {
			t.Errorf("capturedUserID = %q, want %q", capturedUserID, "user-42")
		}
	})

	t.Run("nil slice returns empty JSON array", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return nil, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		if len(dtos) != 0 {
			t.Errorf("len = %d, want 0", len(dtos))
		}
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		if len(dtos) != 2 {
			t.Fatalf("len = %d, want 2", len(dtos))
		}
		if dtos[0].Label != "Alpha" {
			t.Errorf("dtos[0].Label = %q, want %q", dtos[0].Label, "Alpha")
		}
		if dtos[1].Label != "Beta" {
			t.Errorf("dtos[1].Label = %q, want %q", dtos[1].Label, "Beta")
		}
	})
}
