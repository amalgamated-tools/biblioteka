package handlers

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"

	"github.com/stretchr/testify/require"
)

func setupOPDSHandler(t *testing.T) *OPDSHandler {
	t.Helper()
	d := newTestDB(t)
	return &OPDSHandler{DB: d}
}

// parseOPDSFeed parses the response body as an OPDS Atom feed.
func parseOPDSFeed(t *testing.T, body []byte) opdspkg.Feed {
	t.Helper()
	var feed opdspkg.Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		require.Failf(t, "failed", "unmarshal feed: %v\nbody: %s", err, body)
	}
	return feed
}

// findLink returns the first link with the given rel, or nil if not found.
func findLink(links []opdspkg.Link, rel string) *opdspkg.Link {
	for _, l := range links {
		if l.Rel == rel {
			return &l
		}
	}
	return nil
}

// ptr returns a pointer to the given value; used by table-driven tests.
func ptr[T any](v T) *T { return &v }

// --- Routing / method dispatch ---

func TestHandleOPDS_MethodNotAllowed(t *testing.T) {
	h := setupOPDSHandler(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		r := httptest.NewRequest(method, "/opds", nil)
		w := httptest.NewRecorder()
		h.HandleOPDS(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestHandleOPDS_UnknownPath(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
