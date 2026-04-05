package organize

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCrossDeviceRenameError(t *testing.T) {
	t.Run("matches wrapped EXDEV", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", syscall.EXDEV)
		require.True(t, isCrossDeviceRenameError(err))
	})

	t.Run("does not match unrelated errors", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", os.ErrExist)
		require.False(t, isCrossDeviceRenameError(err))
	})

	t.Run("handles nil", func(t *testing.T) {
		require.False(t, isCrossDeviceRenameError(nil))
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
			require.Equal(t, tt.want, got)
		})
	}
}
