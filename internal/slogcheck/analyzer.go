// Package slogcheck provides a go/analysis Analyzer that reports calls to
// [log/slog.Any] whose value argument has a more specific slog attribute
// constructor available (e.g. [log/slog.String] for string values).
//
// slog.Any should only be used when no typed constructor exists — the primary
// legitimate use case is logging error values.  Using slog.Any for primitive
// types (string, int, bool, float64, time.Time, time.Duration, …) bypasses the
// type-safe helpers and makes log attribute types opaque to tooling.
package slogcheck

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the slogcheck analysis.Analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "slogcheck",
	Doc:      "reports slog.Any calls where a typed slog attribute constructor should be used instead",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// typedAlternative maps the string representation of a type to the slog
// constructor that should be used instead of slog.Any.
var typedAlternative = map[string]string{
	"string":        "slog.String",
	"int":           "slog.Int",
	"int64":         "slog.Int64",
	"uint64":        "slog.Uint64",
	"float64":       "slog.Float64",
	"bool":          "slog.Bool",
	"time.Time":     "slog.Time",
	"time.Duration": "slog.Duration",
}

func run(pass *analysis.Pass) (any, error) {
	errorIface := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if !isSlogAny(pass, call) {
			return
		}

		// slog.Any(key, value) — check the value argument (index 1).
		if len(call.Args) < 2 {
			return
		}
		valueArg := call.Args[1]
		t := pass.TypesInfo.TypeOf(valueArg)
		if t == nil {
			return
		}

		// Allow error types — slog.Any is the canonical way to log errors.
		if types.Implements(t, errorIface) {
			return
		}
		// Also allow T when *T implements error (e.g. a non-pointer value
		// whose pointer receiver satisfies the error interface).
		if _, ok := t.(*types.Pointer); !ok && types.Implements(types.NewPointer(t), errorIface) {
			return
		}

		// Normalize untyped basic types (e.g. untyped string from a literal)
		// to their default typed form so the map lookup matches.
		lookup := t
		if basic, ok := t.(*types.Basic); ok && basic.Info()&types.IsUntyped != 0 {
			lookup = types.Default(t)
		}

		// Report if the type has a direct slog constructor.
		if alt, found := typedAlternative[lookup.String()]; found {
			pass.Reportf(call.Pos(), "use %s instead of slog.Any for %s values", alt, lookup)
		}
	})

	return nil, nil
}

// isSlogAny reports whether call is an invocation of the log/slog package-level
// Any function.
func isSlogAny(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Any" {
		return false
	}
	obj := pass.TypesInfo.Uses[sel.Sel]
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	pkg := fn.Pkg()
	return pkg != nil && pkg.Path() == "log/slog"
}
