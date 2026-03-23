package goodreads

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidISBN10CheckDigit(t *testing.T) {
	tests := []struct {
		name  string
		isbn  string
		valid bool
	}{
		{name: "valid plain", isbn: "0306406152", valid: true},
		{name: "valid with hyphens", isbn: "0-306-40615-2", valid: true},
		{name: "valid with X check digit", isbn: "080442957X", valid: true},
		{name: "valid with lowercase x", isbn: "080442957x", valid: true},
		{name: "valid with hyphens and X", isbn: "0-8044-2957-X", valid: true},
		{name: "invalid check digit", isbn: "0306406153", valid: false},
		{name: "too short", isbn: "123456789", valid: false},
		{name: "too long", isbn: "12345678901", valid: false},
		{name: "empty string", isbn: "", valid: false},
		{name: "non-digit characters", isbn: "abcdefghij", valid: false},
		{name: "X not in last position", isbn: "0X06406152", valid: false},
		{name: "valid ISBN - Programming Pearls", isbn: "0201657880", valid: true},
		{name: "valid ISBN with spaces", isbn: "0 306 40615 2", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidISBN10CheckDigit(tt.isbn)
			if got != tt.valid {
				t.Errorf("ValidISBN10CheckDigit(%q) = %v, want %v", tt.isbn, got, tt.valid)
			}
		})
	}
}

func TestValidISBN13CheckDigit(t *testing.T) {
	tests := []struct {
		name  string
		isbn  string
		valid bool
	}{
		{name: "valid plain", isbn: "9780306406157", valid: true},
		{name: "valid with hyphens", isbn: "978-0-306-40615-7", valid: true},
		{name: "valid with spaces", isbn: "978 0 306 40615 7", valid: true},
		{name: "valid ISBN-13 - 979 prefix", isbn: "9791034304721", valid: true},
		{name: "invalid check digit", isbn: "9780306406158", valid: false},
		{name: "too short", isbn: "978030640615", valid: false},
		{name: "too long", isbn: "97803064061577", valid: false},
		{name: "empty string", isbn: "", valid: false},
		{name: "non-digit characters", isbn: "978030640615X", valid: false},
		{name: "letters", isbn: "abcdefghijklm", valid: false},
		{name: "valid - another example", isbn: "9780593135204", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidISBN13CheckDigit(tt.isbn)
			if got != tt.valid {
				t.Errorf("ValidISBN13CheckDigit(%q) = %v, want %v", tt.isbn, got, tt.valid)
			}
		})
	}
}

func TestConvertISBN10To13(t *testing.T) {
	tests := []struct {
		name    string
		isbn10  string
		want    string
		wantErr bool
	}{
		{name: "plain ISBN-10", isbn10: "0306406152", want: "9780306406157"},
		{name: "hyphenated ISBN-10", isbn10: "0-306-40615-2", want: "9780306406157"},
		{name: "ISBN-10 with spaces", isbn10: "0 306 40615 2", want: "9780306406157"},
		{name: "ISBN-10 with X check digit", isbn10: "080442957X", want: "9780804429573"},
		{name: "ISBN-10 with lowercase x", isbn10: "080442957x", want: "9780804429573"},
		{name: "Programming Pearls", isbn10: "0201657880", want: "9780201657883"},
		{name: "invalid check digit", isbn10: "0306406153", wantErr: true},
		{name: "too short", isbn10: "123456789", wantErr: true},
		{name: "empty string", isbn10: "", wantErr: true},
		{name: "non-digit characters", isbn10: "abcdefghij", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertISBN10To13(tt.isbn10)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			// verify the result is a valid ISBN-13
			require.True(t, ValidISBN13CheckDigit(got), "converted ISBN-13 %q failed check digit validation", got)
		})
	}
}
