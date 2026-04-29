package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// errIntentional is a sentinel used to trigger the rollback path in WithTx tests.
var errIntentional = errors.New("intentional rollback")

// TestWithTx_CommitsOnSuccess verifies that a successful fn causes its
// database writes to be committed and visible after WithTx returns.
func TestWithTx_CommitsOnSuccess(t *testing.T) {
	d := newTestDB(t)

	// Insert a tag inside a transaction; fn returns nil → should commit.
	var tagID string
	err := d.WithTx(t.Context(), func(tx *sql.Tx) error {
		var id string
		return tx.QueryRowContext(t.Context(),
			`INSERT INTO tags (name) VALUES ('commit-test') RETURNING id`,
		).Scan(&id)
	})
	require.NoError(t, err, "WithTx returned unexpected error")

	// Verify the row is present outside the transaction.
	err = d.QueryRowContext(t.Context(),
		`SELECT id FROM tags WHERE name = 'commit-test'`,
	).Scan(&tagID)
	require.NoError(t, err, "tag should be visible after successful WithTx")
	require.NotEmpty(t, tagID)
}

// TestWithTx_RollsBackOnError verifies that when fn returns an error the
// transaction is rolled back and no writes are persisted.
func TestWithTx_RollsBackOnError(t *testing.T) {
	d := newTestDB(t)

	// Attempt to insert a tag inside a transaction; fn returns an error → should rollback.
	err := d.WithTx(t.Context(), func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(t.Context(),
			`INSERT INTO tags (name) VALUES ('rollback-test')`,
		)
		if execErr != nil {
			return execErr
		}
		return errIntentional
	})
	require.ErrorIs(t, err, errIntentional, "WithTx should propagate the fn error")

	// Verify the row was rolled back and is not present.
	var id string
	scanErr := d.QueryRowContext(t.Context(),
		`SELECT id FROM tags WHERE name = 'rollback-test'`,
	).Scan(&id)
	require.ErrorIs(t, scanErr, sql.ErrNoRows, "row should not exist after rollback")
}

// TestWithTx_PropagatesFnError verifies that the error returned by fn is
// returned verbatim by WithTx (not wrapped or swapped with a different error).
func TestWithTx_PropagatesFnError(t *testing.T) {
	d := newTestDB(t)

	sentinel := errors.New("my sentinel error")
	err := d.WithTx(t.Context(), func(_ *sql.Tx) error {
		return sentinel
	})
	require.ErrorIs(t, err, sentinel, "WithTx must propagate the exact fn error")
	require.Same(t, sentinel, err, "WithTx must not wrap the fn error")
}

// TestWithTx_NilFnError verifies that a fn that performs no writes and
// returns nil causes WithTx to succeed without error.
func TestWithTx_NilFnError(t *testing.T) {
	d := newTestDB(t)

	err := d.WithTx(t.Context(), func(_ *sql.Tx) error {
		return nil
	})
	require.NoError(t, err, "WithTx with a no-op fn should succeed")
}

// TestDeferRollback_IgnoresErrTxDone verifies that deferRollback does not
// panic or return an error when the transaction has already been committed
// (which causes Rollback to return sql.ErrTxDone).
func TestDeferRollback_IgnoresErrTxDone(t *testing.T) {
	d := newTestDB(t)

	tx, err := d.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	// Commit the transaction first so that the subsequent Rollback returns ErrTxDone.
	require.NoError(t, tx.Commit())

	// deferRollback must not panic.
	require.NotPanics(t, func() {
		deferRollback(t.Context(), tx)
	})
}

// TestDeferRollback_IgnoresAlreadyRolledBack verifies that deferRollback does
// not panic when the transaction has already been rolled back.
func TestDeferRollback_IgnoresAlreadyRolledBack(t *testing.T) {
	d := newTestDB(t)

	tx, err := d.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	// Rollback the transaction explicitly first.
	require.NoError(t, tx.Rollback())

	// A second call via deferRollback must not panic.
	require.NotPanics(t, func() {
		deferRollback(t.Context(), tx)
	})
}
