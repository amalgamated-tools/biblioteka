// Package ollama provides an LLM provider backed by a local Ollama server.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
)

// Client implements llm.Provider using the Ollama /api/chat endpoint.
type Client struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// New creates a new Ollama client with an SSRF-safe HTTP transport.
// baseURL should be e.g. "http://ollama.example.com:11434".
func New(baseURL, model string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		HTTPClient: ssrf.SafeHTTPClient(5 * time.Minute),
	}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// Generate sends a prompt to the Ollama /api/chat endpoint with stream:false
// and returns the model's response content.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaChatRequest{
		Model: c.Model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var result ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}

	if result.Message.Content == "" {
		return "", errors.New("ollama: empty response content")
	}

	return result.Message.Content, nil
}

// Ensure Client implements llm.Provider at compile time.
var _ llm.Provider = (*Client)(nil)
