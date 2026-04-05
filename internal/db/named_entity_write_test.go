package db

import (
	"context"
	"errors"
	"fmt"
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
				t.Error("insertFn should not be called for blank name")
				return nil, nil
			},
		)
		if !errors.Is(err, errFakeInvalid) {
			t.Errorf("namedEntityCreate(%q) = %v, want errFakeInvalid", name, err)
		}
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
	if got.name != want.name {
		t.Errorf("result.name = %q, want %q", got.name, want.name)
	}
}

func TestNamedEntityCreate_UniqueViolationReturnsErrExists(t *testing.T) {
	_, err := namedEntityCreate(t.Context(), "fake", "taken", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, fmt.Errorf("UNIQUE constraint failed: fakes.name")
		},
	)
	if !errors.Is(err, errFakeExists) {
		t.Errorf("namedEntityCreate() = %v, want errFakeExists", err)
	}
}

func TestNamedEntityCreate_PropagatesNonConstraintError(t *testing.T) {
	sentinel := errors.New("disk on fire")
	_, err := namedEntityCreate(t.Context(), "fake", "name", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, sentinel
		},
	)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

// --- namedEntityUpdate ---

func TestNamedEntityUpdate_BlankNameReturnsErrInvalid(t *testing.T) {
	for _, name := range []string{"", "   "} {
		_, err := namedEntityUpdate(t.Context(), "fake", "id-1", name, trimNormalize, errFakeInvalid, errFakeExists,
			func(_ context.Context, _, _ string) (*fakeEntity, error) {
				t.Error("updateFn should not be called for blank name")
				return nil, nil
			},
		)
		if !errors.Is(err, errFakeInvalid) {
			t.Errorf("namedEntityUpdate(%q) = %v, want errFakeInvalid", name, err)
		}
	}
}

func TestNamedEntityUpdate_Success(t *testing.T) {
	got, err := namedEntityUpdate(t.Context(), "fake", "id-1", "updated", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, id, name string) (*fakeEntity, error) {
			if id != "id-1" {
				t.Errorf("updateFn id = %q, want %q", id, "id-1")
			}
			return &fakeEntity{name: name}, nil
		},
	)
	require.NoError(t, err, "namedEntityUpdate() error")
	if got.name != "updated" {
		t.Errorf("result.name = %q, want %q", got.name, "updated")
	}
}

func TestNamedEntityUpdate_UniqueViolationReturnsErrExists(t *testing.T) {
	_, err := namedEntityUpdate(t.Context(), "fake", "id-1", "taken", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _, _ string) (*fakeEntity, error) {
			return nil, fmt.Errorf("UNIQUE constraint failed: fakes.name")
		},
	)
	if !errors.Is(err, errFakeExists) {
		t.Errorf("namedEntityUpdate() = %v, want errFakeExists", err)
	}
}

func TestNamedEntityUpdate_PropagatesNonConstraintError(t *testing.T) {
	sentinel := errors.New("disk on fire")
	_, err := namedEntityUpdate(t.Context(), "fake", "id-1", "name", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _, _ string) (*fakeEntity, error) {
			return nil, sentinel
		},
	)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}
