package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHashHighEntropyTokenWrappers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty",
			token: "",
		},
		{
			name:  "api key",
			token: "bib_1234567890abcdef",
		},
		{
			name:  "kobo token",
			token: "kobo-device-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sum := sha256.Sum256([]byte(tt.token))
			want := hex.EncodeToString(sum[:])

			if got := hashHighEntropyToken(tt.token); got != want {
				t.Fatalf("hashHighEntropyToken(%q) = %q, want %q", tt.token, got, want)
			}
			if got := HashAPIKey(tt.token); got != want {
				t.Fatalf("HashAPIKey(%q) = %q, want %q", tt.token, got, want)
			}
			if got := HashKoboToken(tt.token); got != want {
				t.Fatalf("HashKoboToken(%q) = %q, want %q", tt.token, got, want)
			}
		})
	}
}
