package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// namedEntityCreate and namedEntityUpdate are tested directly here because
// their error-translation branches are easiest to exercise with controlled
// callbacks, mirroring find_or_create_test.go.

// --- namedEntityCreate ---

func TestNamedEntityCreate_BlankNameReturnsErrInvalid(t *testing.T) {
	for _, name := range []string{"", "   "} {
		_, err := namedEntityCreate(t.Context(), "fake", name, trimNormalize, errFakeInvalid, errFakeExists,
			func(_ context.Context, _ string) (*fakeEntity, error) {
				require.Fail(t, "insertFn should not be called for blank name")
				return nil, nil
			},
		)
		require.ErrorIs(t, err, errFakeInvalid)
	}
}

func TestNamedEntityCreate_Success(t *testing.T) {
	want := &fakeEntity{name: "created"}
	got, err := namedEntityCreate(t.Context(), "fake", "created", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, name string) (*fakeEntity, error) {
			return &fakeEntity{name: name}, nil
		},
	)
	require.NoError(t, err, "namedEntityCreate() error")
	require.Equal(t, want.name, got.name)
}

func TestNamedEntityCreate_UniqueViolationReturnsErrExists(t *testing.T) {
	_, err := namedEntityCreate(t.Context(), "fake", "taken", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, errors.New("UNIQUE constraint failed: fakes.name")
		},
	)
	require.ErrorIs(t, err, errFakeExists)
}

func TestNamedEntityCreate_PropagatesNonConstraintError(t *testing.T) {
	sentinel := errors.New("disk on fire")
	_, err := namedEntityCreate(t.Context(), "fake", "name", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, sentinel
		},
	)
	require.ErrorIs(t, err, sentinel)
}

// --- namedEntityUpdate ---

func TestNamedEntityUpdate_BlankNameReturnsErrInvalid(t *testing.T) {
	for _, name := range []string{"", "   "} {
		_, err := namedEntityUpdate(t.Context(), "fake", "id-1", name, trimNormalize, errFakeInvalid, errFakeExists,
			func(_ context.Context, _, _ string) (*fakeEntity, error) {
				require.Fail(t, "updateFn should not be called for blank name")
				return nil, nil
			},
		)
		require.ErrorIs(t, err, errFakeInvalid)
	}
}

func TestNamedEntityUpdate_Success(t *testing.T) {
	got, err := namedEntityUpdate(t.Context(), "fake", "id-1", "updated", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, id, name string) (*fakeEntity, error) {
			require.Equal(t, "id-1", id)
			return &fakeEntity{name: name}, nil
		},
	)
	require.NoError(t, err, "namedEntityUpdate() error")
	require.Equal(t, "updated", got.name)
}

func TestNamedEntityUpdate_UniqueViolationReturnsErrExists(t *testing.T) {
	_, err := namedEntityUpdate(t.Context(), "fake", "id-1", "taken", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _, _ string) (*fakeEntity, error) {
			return nil, errors.New("UNIQUE constraint failed: fakes.name")
		},
	)
	require.ErrorIs(t, err, errFakeExists)
}

func TestNamedEntityUpdate_PropagatesNonConstraintError(t *testing.T) {
	sentinel := errors.New("disk on fire")
	_, err := namedEntityUpdate(t.Context(), "fake", "id-1", "name", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _, _ string) (*fakeEntity, error) {
			return nil, sentinel
		},
	)
	require.ErrorIs(t, err, sentinel)
}
