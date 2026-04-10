package handlers

import (
	"context"
	"errors"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"

	"github.com/stretchr/testify/require"
)

func TestAdaptNavEntities(t *testing.T) {
	type item struct {
		ID   string
		Name string
		Time string
	}

	listFn := func(_ context.Context, limit, offset int) ([]item, int, error) {
		all := []item{
			{ID: "1", Name: "Alpha", Time: "2024-01-01T00:00:00Z"},
			{ID: "2", Name: "Beta", Time: "2024-06-15T12:00:00Z"},
			{ID: "3", Name: "Gamma", Time: "2024-12-31T23:59:59Z"},
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		if offset >= len(all) {
			return nil, len(all), nil
		}
		return all[offset:end], len(all), nil
	}

	toEntity := func(i item) opdspkg.NavEntity {
		return opdspkg.NavEntity{ID: i.ID, Name: i.Name, Updated: i.Time}
	}

	adapted := adaptNavEntities(listFn, toEntity)

	t.Run("converts all items", func(t *testing.T) {
		entities, total, err := adapted(t.Context(), 10, 0)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, entities, 3)
		require.Equal(t, "Alpha", entities[0].Name)
		require.Equal(t, "1", entities[0].ID)
		require.Equal(t, "2024-01-01T00:00:00Z", entities[0].Updated)
		require.Equal(t, "Gamma", entities[2].Name)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		entities, total, err := adapted(t.Context(), 1, 1)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, entities, 1)
		require.Equal(t, "Beta", entities[0].Name)
	})

	t.Run("returns empty slice for offset past end", func(t *testing.T) {
		entities, total, err := adapted(t.Context(), 10, 100)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Empty(t, entities)
	})

	t.Run("propagates errors", func(t *testing.T) {
		errBoom := errors.New("boom")
		failingList := func(_ context.Context, _, _ int) ([]item, int, error) {
			return nil, 0, errBoom
		}
		failAdapted := adaptNavEntities(failingList, toEntity)

		_, _, err := failAdapted(t.Context(), 10, 0)
		require.ErrorIs(t, err, errBoom)
	})
}
