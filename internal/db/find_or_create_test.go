package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// findOrCreate is tested directly here because its race-condition branch and
// error propagation are easiest to exercise with controlled callbacks.

var (
	errFakeInvalid = errors.New("test: invalid name")
	errFakeExists  = errors.New("test: already exists")
)

type fakeEntity struct {
	name string
}

func identityNormalize(s string) string { return s }

func trimNormalize(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b != ' ' {
			result = append(result, b)
		}
	}
	return string(result)
}

func TestFindOrCreate_BlankNameReturnsErrInvalid(t *testing.T) {
	for _, name := range []string{"", "   "} {
		_, err := findOrCreate(t.Context(), name, "fake", trimNormalize, errFakeInvalid, errFakeExists,
			func(_ context.Context, _ string) (*fakeEntity, error) {
				t.Error("getByName should not be called for blank name")
				return nil, nil
			},
			func(_ context.Context, _ string) (*fakeEntity, error) {
				t.Error("create should not be called for blank name")
				return nil, nil
			},
		)
		if !errors.Is(err, errFakeInvalid) {
			t.Errorf("findOrCreate(%q) = %v, want errFakeInvalid", name, err)
		}
	}
}

func TestFindOrCreate_CreatesNewEntity(t *testing.T) {
	store := map[string]*fakeEntity{}

	result, err := findOrCreate(t.Context(), "newEntity", "fake", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, name string) (*fakeEntity, error) {
			if e, ok := store[name]; ok {
				return e, nil
			}
			return nil, sql.ErrNoRows
		},
		func(_ context.Context, name string) (*fakeEntity, error) {
			if _, ok := store[name]; ok {
				return nil, errFakeExists
			}
			e := &fakeEntity{name: name}
			store[name] = e
			return e, nil
		},
	)
	require.NoError(t, err, "findOrCreate() error")
	if result == nil {
		require.Fail(t, "findOrCreate() returned nil")
	}
	if result.name != "newEntity" {
		t.Errorf("result.name = %q, want newEntity", result.name)
	}
}

func TestFindOrCreate_ReturnsExistingEntity(t *testing.T) {
	existing := &fakeEntity{name: "existing"}
	store := map[string]*fakeEntity{"existing": existing}

	result, err := findOrCreate(t.Context(), "existing", "fake", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, name string) (*fakeEntity, error) {
			if e, ok := store[name]; ok {
				return e, nil
			}
			return nil, sql.ErrNoRows
		},
		func(_ context.Context, name string) (*fakeEntity, error) {
			e := &fakeEntity{name: name}
			store[name] = e
			return e, nil
		},
	)
	require.NoError(t, err, "findOrCreate() error")
	if result != existing {
		t.Errorf("findOrCreate() returned different entity, want original")
	}
}

// TestFindOrCreate_RaceCondition simulates a concurrent insert: getByName
// initially returns sql.ErrNoRows, create returns errFakeExists (someone else
// won the race), and then the second getByName call returns the winner's
// entity.
func TestFindOrCreate_RaceCondition(t *testing.T) {
	winner := &fakeEntity{name: "raced"}
	callCount := 0

	result, err := findOrCreate(t.Context(), "raced", "fake", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			callCount++
			if callCount == 1 {
				// First call: entity not yet visible.
				return nil, sql.ErrNoRows
			}
			// Second call (after failed insert): entity now exists.
			return winner, nil
		},
		func(_ context.Context, _ string) (*fakeEntity, error) {
			// Simulate losing the race (unique constraint violation).
			return nil, errFakeExists
		},
	)
	require.NoError(t, err, "findOrCreate() error")
	if result != winner {
		t.Errorf("findOrCreate() did not return the race winner")
	}
	if callCount != 2 {
		t.Errorf("getByName called %d times, want 2", callCount)
	}
}

func TestFindOrCreate_PropagatesGetByNameError(t *testing.T) {
	sentinel := errors.New("database unavailable")

	_, err := findOrCreate(t.Context(), "name", "fake", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, sentinel
		},
		func(_ context.Context, _ string) (*fakeEntity, error) {
			t.Error("create should not be called when getByName returns non-ErrNoRows")
			return nil, nil
		},
	)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

func TestFindOrCreate_PropagatesCreateError(t *testing.T) {
	createErr := errors.New("create failed unexpectedly")

	_, err := findOrCreate(t.Context(), "name", "fake", identityNormalize, errFakeInvalid, errFakeExists,
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, sql.ErrNoRows
		},
		func(_ context.Context, _ string) (*fakeEntity, error) {
			return nil, createErr
		},
	)
	if !errors.Is(err, createErr) {
		t.Errorf("err = %v, want %v", err, createErr)
	}
}

// TestFindOrCreate_ViaFindOrCreateAuthor exercises the full code path through
// a real DB call, ensuring FindOrCreateAuthor integration works end-to-end.
func TestFindOrCreate_ViaFindOrCreateAuthor(t *testing.T) {
	d := newTestDB(t)

	// First call creates.
	a1, err := d.FindOrCreateAuthor(t.Context(), "Neil Gaiman")
	require.NoError(t, err, "first FindOrCreateAuthor() error")
	if a1.ID == "" {
		t.Error("created author has empty ID")
	}

	// Second call finds the same record.
	a2, err := d.FindOrCreateAuthor(t.Context(), "Neil Gaiman")
	require.NoError(t, err, "second FindOrCreateAuthor() error")
	if a2.ID != a1.ID {
		t.Errorf("IDs differ: %q vs %q", a2.ID, a1.ID)
	}
}
