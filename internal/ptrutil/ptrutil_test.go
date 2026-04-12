package ptrutil_test

import (
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/ptrutil"
	"github.com/stretchr/testify/require"
)

func TestDeref(t *testing.T) {
	t.Run("non-nil string", func(t *testing.T) {
		s := "hello"
		require.Equal(t, "hello", ptrutil.Deref(&s))
	})
	t.Run("nil string", func(t *testing.T) {
		require.Equal(t, "", ptrutil.Deref((*string)(nil)))
	})
	t.Run("non-nil int64", func(t *testing.T) {
		v := int64(99)
		require.Equal(t, int64(99), ptrutil.Deref(&v))
	})
	t.Run("nil int64", func(t *testing.T) {
		require.Equal(t, int64(0), ptrutil.Deref((*int64)(nil)))
	})
	t.Run("nil float64", func(t *testing.T) {
		require.InDelta(t, 0.0, ptrutil.Deref((*float64)(nil)), 0.001)
	})
}

func TestNilIfZero(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		p := ptrutil.NilIfZero("hello")
		require.NotNil(t, p)
		require.Equal(t, "hello", *p)
	})
	t.Run("empty string", func(t *testing.T) {
		require.Nil(t, ptrutil.NilIfZero(""))
	})
	t.Run("non-zero int64", func(t *testing.T) {
		p := ptrutil.NilIfZero(int64(7))
		require.NotNil(t, p)
		require.Equal(t, int64(7), *p)
	})
	t.Run("zero int64", func(t *testing.T) {
		require.Nil(t, ptrutil.NilIfZero(int64(0)))
	})
	t.Run("non-zero float64", func(t *testing.T) {
		p := ptrutil.NilIfZero(3.14)
		require.NotNil(t, p)
		require.InDelta(t, 3.14, *p, 0.001)
	})
	t.Run("zero float64", func(t *testing.T) {
		require.Nil(t, ptrutil.NilIfZero(0.0))
	})
}
