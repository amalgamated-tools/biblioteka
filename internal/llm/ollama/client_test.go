package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_Generate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: `{"genres":["Fiction"]}`},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	content, err := client.Generate(t.Context(), "test prompt")
	require.NoError(t, err)
	require.Equal(t, `{"genres":["Fiction"]}`, content)
}

func TestClient_Generate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	_, err := client.Generate(t.Context(), "test prompt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status 500")
}

func TestClient_Generate_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: ""},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	_, err := client.Generate(t.Context(), "test prompt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty response content")
}
