package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashHighEntropyTokenWrappers(t *testing.T) {
	t.Parallel()

	// Golden values precomputed with: printf '<token>' | sha256sum
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "empty",
			token: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "api key",
			token: "bib_1234567890abcdef",
			want:  "e5f5aec6f19714354a47cf5f642057e5277b65883221ea66c12cb56407750ea6",
		},
		{
			name:  "kobo token",
			token: "kobo-device-token",
			want:  "850c11e72df3e537835f92b3e259ef4fff7c803a41d72fee558e23b559ce418b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := hashHighEntropyToken(tt.token); got != tt.want {
				require.Failf(t, "failed", "hashHighEntropyToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
			if got := HashAPIKey(tt.token); got != tt.want {
				require.Failf(t, "failed", "HashAPIKey(%q) = %q, want %q", tt.token, got, tt.want)
			}
			if got := HashKoboToken(tt.token); got != tt.want {
				require.Failf(t, "failed", "HashKoboToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}
