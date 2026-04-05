// Package testutils provides helper functions for constructing synthetic book
// files (EPUB, MOBI, PDF) in tests. These helpers create minimal but valid
// file structures so that metadata extraction and processing pipelines can be
// exercised without real book assets.
//
// All helpers accept a *testing.T and call t.Fatal on errors so tests fail
// immediately with a clear message. The package must only be imported by
// _test.go files.
package testutils
