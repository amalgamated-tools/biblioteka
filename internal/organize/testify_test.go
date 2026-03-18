package organize

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func fail(t testing.TB, failureMessage string, msgAndArgs ...any) {
	t.Helper()
	require.Fail(t, failureMessage, msgAndArgs...)
}

func failf(t testing.TB, format string, args ...any) {
	t.Helper()
	require.Failf(t, "assertion failed", format, args...)
}

func failNow(t testing.TB, failureMessage string, msgAndArgs ...any) {
	t.Helper()
	require.FailNow(t, failureMessage, msgAndArgs...)
}

func failNowf(t testing.TB, format string, args ...any) {
	t.Helper()
	require.FailNowf(t, "assertion failed", format, args...)
}
