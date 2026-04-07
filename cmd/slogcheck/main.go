// Command slogcheck runs the slogcheck analyzer over the provided packages.
// Usage:
//
//	go run ./cmd/slogcheck ./...
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/amalgamated-tools/biblioteka/internal/slogcheck"
)

func main() {
	singlechecker.Main(slogcheck.Analyzer)
}
