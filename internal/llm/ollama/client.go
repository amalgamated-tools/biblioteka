// Package ollama provides an LLM provider backed by a local Ollama server.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/llm"
)

// privateIPNets lists IP networks that must never be targeted by an Ollama
// request (SSRF protection).
var privateIPNets = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC-1918 class A private
		"172.16.0.0/12",  // RFC-1918 class B private
		"192.168.0.0/16", // RFC-1918 class C private
		"127.0.0.0/8",    // IPv4 loopback
		"169.254.0.0/16", // IPv4 link-local (incl. AWS IMDS 169.254.169.254)
		"0.0.0.0/8",      // "this" network
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique-local (fc00:: / fd00::)
		"fe80::/10",      // IPv6 link-local
		"::/128",         // IPv6 unspecified
		"100.64.0.0/10",  // Shared address space (RFC 6598)
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("ollama: invalid CIDR %q: %v", cidr, err))
		}
		nets = append(nets, n)
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateIPNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ssrfSafeHTTPClient returns an *http.Client whose dialer validates that the
// resolved IP address is not in a private/loopback/link-local range. This
// prevents DNS rebinding attacks where a hostname passes the initial
// validation but resolves to a private address at connect time.
func ssrfSafeHTTPClient() *http.Client {
	baseDialer := &net.Dialer{}
	safeDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		var safeIP string
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if isPrivateIP(ip) {
				return nil, fmt.Errorf("refusing to connect to private address %s", ipStr)
			}
			if safeIP == "" {
				safeIP = ipStr
			}
		}
		if safeIP == "" {
			return nil, fmt.Errorf("no valid addresses for host %s", host)
		}
		// Connect directly to the validated IP — never re-resolve the hostname.
		return baseDialer.DialContext(ctx, network, net.JoinHostPort(safeIP, port))
	}
	return &http.Client{
		Timeout:   5 * time.Minute,
		Transport: &http.Transport{DialContext: safeDialContext},
	}
}

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
		HTTPClient: ssrfSafeHTTPClient(),
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
