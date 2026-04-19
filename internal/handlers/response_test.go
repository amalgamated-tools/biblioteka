package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteSecretTokenResponse_HeadersAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"token": "supersecret"}

	writeSecretTokenResponse(t.Context(), w, http.StatusCreated, data)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", w.Header().Get("Pragma"))
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Equal(t, "supersecret", result["token"])
}

func TestWriteSecretTokenResponse_200OK(t *testing.T) {
	w := httptest.NewRecorder()
	writeSecretTokenResponse(t.Context(), w, http.StatusOK, map[string]string{"key": "value"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", w.Header().Get("Pragma"))
}
