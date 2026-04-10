// Package p contains errorfcheck test fixtures.
package p

import (
	"errors"
	"fmt"
)

func examples() {
	// Static strings — all should be flagged.
	_ = fmt.Errorf("something went wrong")        // want `fmt\.Errorf with no format verbs; use errors\.New instead`
	_ = fmt.Errorf("connection refused")          // want `fmt\.Errorf with no format verbs; use errors\.New instead`
	_ = fmt.Errorf("EOF reached")                 // want `fmt\.Errorf with no format verbs; use errors\.New instead`
	_ = fmt.Errorf("a 50%% literal percent sign") // want `fmt\.Errorf with no format verbs; use errors\.New instead`

	// Static string with extra arguments — should suggest adding verbs, not errors.New.
	_ = fmt.Errorf("something failed", errors.New("x")) // want `fmt\.Errorf format string has no verbs but has extra arguments; add format verbs \(e\.g\. %w\) or remove extra arguments`

	// Raw string literal — should also be flagged.
	_ = fmt.Errorf(`static backtick message`) // want `fmt\.Errorf with no format verbs; use errors\.New instead`

	// Format verbs present — must not be flagged.
	_ = fmt.Errorf("failed: %w", errors.New("x"))
	_ = fmt.Errorf("got %d items", 5)
	_ = fmt.Errorf("value: %v", "x")
	_ = fmt.Errorf("name: %s", "alice")
	_ = fmt.Errorf("50%% done, %d remaining", 3)

	// Non-literal format string — must not be flagged (can't analyze at
	// compile time).
	msg := "dynamic"
	_ = fmt.Errorf(msg)
}

const staticMsg = "constant error message"

func constantExample() {
	// Named string constant with no verbs — should be flagged.
	_ = fmt.Errorf(staticMsg) // want `fmt\.Errorf with no format verbs; use errors\.New instead`
}
