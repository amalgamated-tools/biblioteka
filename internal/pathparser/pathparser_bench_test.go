package pathparser

import "testing"

// BenchmarkParseBookPath measures the cost of the full path-parsing pipeline
// across representative library layouts (3-level series, 2-level author, flat
// dash-separated, and 0-segment single file).  These benchmarks are run in CI
// to catch regressions in parsing performance.

func BenchmarkParseBookPath_ThreeLevel(b *testing.B) {
	filePath := "/library/Brandon Sanderson/Mistborn/2. The Well of Ascension (2006).epub"
	root := "/library"
	b.ReportAllocs()
	for range b.N {
		_ = ParseBookPath(filePath, root)
	}
}

func BenchmarkParseBookPath_TwoLevel(b *testing.B) {
	filePath := "/library/Agatha Christie/1. The Seven Dials Mystery - Agatha Christie (2010).epub"
	root := "/library"
	b.ReportAllocs()
	for range b.N {
		_ = ParseBookPath(filePath, root)
	}
}

func BenchmarkParseBookPath_FlatDash(b *testing.B) {
	filePath := "/library/The Name of the Wind - Patrick Rothfuss.epub"
	root := "/library"
	b.ReportAllocs()
	for range b.N {
		_ = ParseBookPath(filePath, root)
	}
}

func BenchmarkParseBookPath_SingleFile(b *testing.B) {
	filePath := "/library/dune.epub"
	root := "/library"
	b.ReportAllocs()
	for range b.N {
		_ = ParseBookPath(filePath, root)
	}
}
