package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogout_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	for _, method := range []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
	} {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "/api/auth/logout", nil)
			w := httptest.NewRecorder()
			h.Logout(w, r)
			require.Equal(t, http.StatusMethodNotAllowed, w.Code,
				"%s to /api/auth/logout should return 405", method)
		})
	}
}
