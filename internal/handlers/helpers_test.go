package handlers

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// mustMarshal marshals v to JSON and fails the test if marshaling fails.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err, "mustMarshal")
	return data
}

func Test_WriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(t.Context(), w, http.StatusOK, map[string]string{"key": "value"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var result map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
	require.Equal(t, "value", result["key"])
}

func Test_WriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(t.Context(), w, http.StatusBadRequest, "something went wrong")

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var result map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
	require.Equal(t, "something went wrong", result["error"])
}

func Test_LogAudit(t *testing.T) {
	d := newTestDB(t)

	logAudit(t.Context(), d, "user-42", db.AuditActionBookCreated, "book", "book-1", map[string]any{"title": "Audited"})

	logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].UserID)
	require.Equal(t, "user-42", *logs[0].UserID)
	require.Equal(t, db.AuditActionBookCreated, logs[0].Action)
	require.Equal(t, "book", logs[0].EntityType)
	require.Equal(t, "book-1", logs[0].EntityID)
	require.NotNil(t, logs[0].Metadata, "metadata = nil, want JSON metadata")
}

func TestRequestScheme(t *testing.T) {
	tests := []struct {
		name   string
		tls    bool
		header string // X-Forwarded-Proto value
		want   string
	}{
		{name: "plain HTTP", tls: false, header: "", want: "http"},
		{name: "TLS connection", tls: true, header: "", want: "https"},
		{name: "forwarded https", tls: false, header: "https", want: "https"},
		{name: "forwarded http", tls: true, header: "http", want: "http"},
		{name: "uppercase HTTPS", tls: false, header: "HTTPS", want: "https"},
		{name: "mixed case Https", tls: false, header: "Https", want: "https"},
		{name: "padded whitespace", tls: false, header: "  https  ", want: "https"},
		{name: "invalid proto ignored", tls: false, header: "javascript:", want: "http"},
		{name: "ftp proto ignored", tls: false, header: "ftp", want: "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			if tt.header != "" {
				r.Header.Set("X-Forwarded-Proto", tt.header)
			}
			got := requestScheme(r)
			require.Equal(t, tt.want, got)
		})
	}
}
