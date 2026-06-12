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

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
		require.Equal(t, "failed to list widgets", result["error"])
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		require.Len(t, dtos, 2, "len(dtos)")
		require.Equal(t, "Alpha", dtos[0].Label)
		require.Equal(t, "Beta", dtos[1].Label)
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		require.Len(t, dtos, 0)
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
		require.Len(t, result, 2, "len(result)")
		require.Equal(t, "Alpha", result[0].Label)
		require.Equal(t, "Beta", result[1].Label)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := mapSlice([]entity{}, toDTO)
		require.NotNil(t, result, "result should not be nil")
		require.Len(t, result, 0)
	})

	t.Run("nil input returns empty slice", func(t *testing.T) {
		result := mapSlice(nil, toDTO)
		require.NotNil(t, result, "result should not be nil")
		require.Len(t, result, 0)
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

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
		require.Equal(t, "failed to list tokens", result["error"])
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

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "user-42", capturedUserID)
	})

	t.Run("nil slice returns empty JSON array", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return nil, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		require.Len(t, dtos, 0)
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []dto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "failed to unmarshal")
		require.Len(t, dtos, 2, "len(dtos)")
		require.Equal(t, "Alpha", dtos[0].Label)
		require.Equal(t, "Beta", dtos[1].Label)
	})
}

func Test_ListPaginatedEntities(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string `json:"label"`
	}
	type listDTO struct {
		Items []dto `json:"items"`
		paginationMeta
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}
	makeListDTO := func(items []dto, total, limit, offset int) listDTO {
		return listDTO{
			Items: items,
			paginationMeta: paginationMeta{
				Total:  total,
				Limit:  limit,
				Offset: offset,
			},
		}
	}

	t.Run("error yields 500", func(t *testing.T) {
		listFn := func(_ context.Context, _, _ int) ([]entity, int, error) {
			return nil, 0, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listPaginatedEntities(w, r, "widgets", listFn, toDTO, makeListDTO)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
		require.Equal(t, "failed to list widgets", result["error"])
	})

	t.Run("passes parsed pagination values", func(t *testing.T) {
		var capturedLimit, capturedOffset int
		listFn := func(_ context.Context, limit, offset int) ([]entity, int, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []entity{}, 0, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?limit=7&offset=3", nil)
		listPaginatedEntities(w, r, "widgets", listFn, toDTO, makeListDTO)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 7, capturedLimit)
		require.Equal(t, 3, capturedOffset)
	})

	t.Run("success converts items and includes pagination", func(t *testing.T) {
		listFn := func(_ context.Context, _, _ int) ([]entity, int, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, 12, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?limit=2&offset=4", nil)
		listPaginatedEntities(w, r, "widgets", listFn, toDTO, makeListDTO)

		require.Equal(t, http.StatusOK, w.Code)
		var response listDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response), "failed to unmarshal")
		require.Len(t, response.Items, 2)
		require.Equal(t, "Alpha", response.Items[0].Label)
		require.Equal(t, "Beta", response.Items[1].Label)
		require.Equal(t, 12, response.Total)
		require.Equal(t, 2, response.Limit)
		require.Equal(t, 4, response.Offset)
	})
}
