package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// HandleAuth handles POST /v1/auth/device, /v1/auth/refresh, and /v1/auth/exchange.
// The Kobo device doesn't use these tokens for our server, so we return dummy values.
func (h *KoboHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	var userKey string
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<16) // 64 KiB is ample for an auth payload
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if k, ok := body["UserKey"].(string); ok {
				userKey = k
			}
		}
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "internal error")
		return
	}
	accessToken := base64.StdEncoding.EncodeToString(buf)
	if _, err := rand.Read(buf); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "internal error")
		return
	}
	refreshToken := base64.StdEncoding.EncodeToString(buf)

	writeKoboJSON(w, http.StatusOK, map[string]any{
		"AccessToken":  accessToken,
		"RefreshToken": refreshToken,
		"TokenType":    "Bearer",
		"TrackingId":   koboRandomUUID(),
		"UserKey":      userKey,
	})
}

// koboRandomUUID generates a random UUID v4-like string.
func koboRandomUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
