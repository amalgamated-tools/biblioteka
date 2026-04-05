package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_HandleNameErr(t *testing.T) {
	var (
		errInvalid = errors.New("invalid name")
		errExists  = errors.New("name exists")
		errOther   = errors.New("other error")
	)

	tests := []struct {
		name        string
		err         error
		resourceArt string
		wantHandled bool
		wantCode    int
		wantErrMsg  string
	}{
		{
			name:        "nil error not handled",
			err:         nil,
			resourceArt: "an author",
			wantHandled: false,
		},
		{
			name:        "errInvalid yields 400",
			err:         errInvalid,
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "wrapped errInvalid yields 400",
			err:         fmt.Errorf("db: %w", errInvalid),
			resourceArt: "a series",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "wrapped errExists yields 409",
			err:         fmt.Errorf("db: %w", errExists),
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "an author with that name already exists",
		},
		{
			name:        "errExists yields 409",
			err:         errExists,
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "an author with that name already exists",
		},
		{
			name:        "errExists series yields 409",
			err:         errExists,
			resourceArt: "a series",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "a series with that name already exists",
		},
		{
			name:        "unrelated error not handled",
			err:         errOther,
			resourceArt: "an author",
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleNameErr(t.Context(), w, tt.err, errInvalid, errExists, tt.resourceArt)
			require.Equal(t, tt.wantHandled, got, "handleNameErr()")
			if !tt.wantHandled {
				if w.Code != http.StatusOK {
					t.Errorf("expected no response written, but got status %d", w.Code)
				}
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body, but got %q", w.Body.String())
				}
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			if result["error"] != tt.wantErrMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantErrMsg)
			}
		})
	}
}

func Test_HandleDBErr(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		resource    string
		wantHandled bool
		wantCode    int
		wantMsg     string
	}{
		{
			name:        "nil error",
			err:         nil,
			resource:    "book",
			wantHandled: false,
		},
		{
			name:        "not found",
			err:         sql.ErrNoRows,
			resource:    "author",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantMsg:     "author not found",
		},
		{
			name:        "wrapped not found",
			err:         fmt.Errorf("lookup failed: %w", sql.ErrNoRows),
			resource:    "series",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantMsg:     "series not found",
		},
		{
			name:        "other error",
			err:         errors.New("connection refused"),
			resource:    "library",
			wantHandled: true,
			wantCode:    http.StatusInternalServerError,
			wantMsg:     "failed to get library",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleDBErr(t.Context(), w, tt.err, tt.resource)
			require.Equal(t, tt.wantHandled, got, "handleDBErr()")
			if !tt.wantHandled {
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			if result["error"] != tt.wantMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantMsg)
			}
		})
	}
}

func Test_HandleUpdateErr(t *testing.T) {
	var (
		errInvalid = errors.New("invalid name")
		errExists  = errors.New("name exists")
		errOther   = errors.New("other error")
	)

	tests := []struct {
		name        string
		err         error
		resourceArt string
		resource    string
		id          string
		wantHandled bool
		wantCode    int
		wantErrMsg  string
	}{
		{
			name:        "nil error not handled",
			err:         nil,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-1",
			wantHandled: false,
		},
		{
			name:        "not found yields 404",
			err:         sql.ErrNoRows,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-1",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantErrMsg:  "author not found",
		},
		{
			name:        "wrapped not found yields 404",
			err:         fmt.Errorf("db: %w", sql.ErrNoRows),
			resourceArt: "a series",
			resource:    "series",
			id:          "ser-1",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantErrMsg:  "series not found",
		},
		{
			name:        "invalid name yields 400",
			err:         errInvalid,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-2",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "duplicate name yields 409",
			err:         errExists,
			resourceArt: "a series",
			resource:    "series",
			id:          "ser-2",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "a series with that name already exists",
		},
		{
			name:        "other error yields 500",
			err:         errOther,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-3",
			wantHandled: true,
			wantCode:    http.StatusInternalServerError,
			wantErrMsg:  "failed to update author",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleUpdateErr(t.Context(), w, tt.err, errInvalid, errExists, tt.resourceArt, tt.resource, tt.id)
			require.Equal(t, tt.wantHandled, got, "handleUpdateErr()")
			if !tt.wantHandled {
				if w.Code != http.StatusOK {
					t.Errorf("expected no response written, but got status %d", w.Code)
				}
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body, but got %q", w.Body.String())
				}
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			if result["error"] != tt.wantErrMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantErrMsg)
			}
		})
	}
}
