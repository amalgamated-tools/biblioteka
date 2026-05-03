package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLimitOffset(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		defaultLimit int
		maxLimit     int
		wantLimit    int
		wantOffset   int
	}{
		{
			name:         "no params uses default limit",
			url:          "/",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "valid limit",
			url:          "/?limit=10",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    10,
			wantOffset:   0,
		},
		{
			name:         "valid limit and offset",
			url:          "/?limit=20&offset=5",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    20,
			wantOffset:   5,
		},
		{
			name:         "limit exceeds max is clamped to max",
			url:          "/?limit=999",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    200,
			wantOffset:   0,
		},
		{
			name:         "limit=0 uses default (below minimum of 1)",
			url:          "/?limit=0",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "negative limit uses default",
			url:          "/?limit=-5",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "non-integer limit uses default",
			url:          "/?limit=abc",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "negative offset clamps to 0",
			url:          "/?offset=-1",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "non-integer offset uses 0",
			url:          "/?offset=xyz",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "limit=1 is minimum valid",
			url:          "/?limit=1",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    1,
			wantOffset:   0,
		},
		{
			name:         "limit equals max is accepted",
			url:          "/?limit=200",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    200,
			wantOffset:   0,
		},
		{
			name:         "offset=0 is valid",
			url:          "/?offset=0",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "large valid offset",
			url:          "/?offset=1000",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   1000,
		},
		{
			name:         "offset exceeding max is clamped",
			url:          "/?offset=99999999",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   maxPageOffset,
		},
		{
			name:         "custom default and max limits",
			url:          "/",
			defaultLimit: 25,
			maxLimit:     100,
			wantLimit:    25,
			wantOffset:   0,
		},
		{
			name:         "limit above custom max clamped",
			url:          "/?limit=150",
			defaultLimit: 25,
			maxLimit:     100,
			wantLimit:    100,
			wantOffset:   0,
		},
		{
			name:         "empty limit param uses default",
			url:          "/?limit=",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "empty offset param uses 0",
			url:          "/?offset=",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "limit overflow uses default",
			url:          "/?limit=99999999999999999999",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "offset overflow uses 0",
			url:          "/?offset=99999999999999999999",
			defaultLimit: 50,
			maxLimit:     200,
			wantLimit:    50,
			wantOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			gotLimit, gotOffset := parseLimitOffset(r, tt.defaultLimit, tt.maxLimit)
			require.Equal(t, tt.wantLimit, gotLimit)
			require.Equal(t, tt.wantOffset, gotOffset)
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		defaultLimit int
		maxLimit     int
		wantLimit    int
	}{
		{
			name:         "no params uses default limit",
			url:          "/",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
		{
			name:         "valid limit",
			url:          "/?limit=5",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    5,
		},
		{
			name:         "limit exceeds max is clamped",
			url:          "/?limit=999",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    50,
		},
		{
			name:         "limit=0 uses default",
			url:          "/?limit=0",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
		{
			name:         "negative limit uses default",
			url:          "/?limit=-1",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
		{
			name:         "non-integer limit uses default",
			url:          "/?limit=abc",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
		{
			name:         "offset param is ignored",
			url:          "/?limit=5&offset=100",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    5,
		},
		{
			name:         "empty limit param uses default",
			url:          "/?limit=",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
		{
			name:         "limit equals max is accepted",
			url:          "/?limit=50",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    50,
		},
		{
			name:         "limit overflow uses default",
			url:          "/?limit=99999999999999999999",
			defaultLimit: 10,
			maxLimit:     50,
			wantLimit:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			gotLimit := parseLimit(r, tt.defaultLimit, tt.maxLimit)
			require.Equal(t, tt.wantLimit, gotLimit)
		})
	}
}
