// Package p contains slogcheck test fixtures.
package p

import (
	"errors"
	"log/slog"
	"time"
)

// errorValue implements the error interface.
type errorValue struct{ msg string }

func (e *errorValue) Error() string { return e.msg }

// myStruct is a complex type with no direct slog equivalent.
type myStruct struct{ X int }

func examples() {
	var (
		s   string
		i   int
		i64 int64
		u64 uint64
		f64 float64
		b   bool
		t   time.Time
		d   time.Duration
	)

	// Primitive types — all should be flagged.
	slog.Any("key", s)   // want `use slog\.String instead of slog\.Any for string values`
	slog.Any("key", i)   // want `use slog\.Int instead of slog\.Any for int values`
	slog.Any("key", i64) // want `use slog\.Int64 instead of slog\.Any for int64 values`
	slog.Any("key", u64) // want `use slog\.Uint64 instead of slog\.Any for uint64 values`
	slog.Any("key", f64) // want `use slog\.Float64 instead of slog\.Any for float64 values`
	slog.Any("key", b)   // want `use slog\.Bool instead of slog\.Any for bool values`
	slog.Any("key", t)   // want `use slog\.Time instead of slog\.Any for time\.Time values`
	slog.Any("key", d)   // want `use slog\.Duration instead of slog\.Any for time\.Duration values`

	// Untyped literals — should also be flagged.
	slog.Any("key", "value") // want `use slog\.String instead of slog\.Any for string values`
	slog.Any("key", 1)       // want `use slog\.Int instead of slog\.Any for int values`
	slog.Any("key", true)    // want `use slog\.Bool instead of slog\.Any for bool values`
	slog.Any("key", 3.14)    // want `use slog\.Float64 instead of slog\.Any for float64 values`

	// Untyped constant — should also be flagged.
	const untypedStr = "hello"
	slog.Any("key", untypedStr) // want `use slog\.String instead of slog\.Any for string values`

	// error values — must not be flagged.
	err := errors.New("oops")
	slog.Any("error", err)
	var ev *errorValue
	slog.Any("error", ev)
	// Non-pointer value where *T implements error — must not be flagged.
	var evVal errorValue
	slog.Any("error", evVal)

	// Complex struct — no direct slog equivalent, must not be flagged.
	slog.Any("struct", &myStruct{X: 1})
}
