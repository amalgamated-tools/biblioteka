package errorfcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/amalgamated-tools/biblioteka/internal/errorfcheck"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errorfcheck.Analyzer, "p")
}
