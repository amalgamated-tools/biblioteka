// Package ptrutil provides small, generic pointer-conversion helpers.
package ptrutil

// Ptr returns a pointer to v. This is useful for taking the address of
// literals, constants, or return values that cannot be addressed directly.
func Ptr[T any](v T) *T { return &v }

// Deref safely dereferences p, returning the zero value of T when p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// NilIfZero returns a pointer to v, or nil when v equals its zero value.
// Useful for mapping "empty" values (empty string, zero int, …) to SQL NULLs
// or optional JSON fields.
func NilIfZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
