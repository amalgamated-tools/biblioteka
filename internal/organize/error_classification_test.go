package organize

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
)

func TestIsCrossDeviceRenameError(t *testing.T) {
	t.Run("matches wrapped EXDEV", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", syscall.EXDEV)
		if !isCrossDeviceRenameError(err) {
			t.Fatal("expected wrapped EXDEV to be classified as cross-device")
		}
	})

	t.Run("does not match unrelated errors", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", os.ErrExist)
		if isCrossDeviceRenameError(err) {
			t.Fatal("expected os.ErrExist not to be classified as cross-device")
		}
	})

	t.Run("handles nil", func(t *testing.T) {
		if isCrossDeviceRenameError(nil) {
			t.Fatal("expected nil not to be classified as cross-device")
		}
	})
}

func TestIsBenignCleanupRemoveError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "wrapped not exist",
			err:  fmt.Errorf("wrapped: %w", os.ErrNotExist),
			want: true,
		},
		{
			name: "wrapped enotempty",
			err:  fmt.Errorf("wrapped: %w", syscall.ENOTEMPTY),
			want: true,
		},
		{
			name: "unrelated",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBenignCleanupRemoveError(tt.err)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
