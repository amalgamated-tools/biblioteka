package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimestamp_Scan_TimeValue(t *testing.T) {
	now := time.Now().UTC()
	var ts Timestamp
	require.NoError(t, ts.Scan(now), "Scan(time.Time) error")
	if !ts.Equal(now) {
		t.Errorf("got %v, want %v", ts.Time, now)
	}
}

func TestTimestamp_Scan_TimeWithTimezone(t *testing.T) {
	loc := time.FixedZone("EST", -5*60*60)
	est := time.Date(2026, 2, 24, 10, 30, 0, 0, loc)
	var ts Timestamp
	require.NoError(t, ts.Scan(est), "Scan(time.Time with timezone) error")
	if !ts.Equal(est) {
		t.Errorf("got %v, want %v", ts.Time, est)
	}
}

func TestTimestamp_Scan_Nil(t *testing.T) {
	var ts Timestamp
	ts.Time = time.Now() // set non-zero first
	require.NoError(t, ts.Scan(nil), "Scan(nil) error")
	if !ts.IsZero() {
		t.Errorf("expected zero time after scanning nil, got %v", ts.Time)
	}
}

func TestTimestamp_Scan_StringFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"RFC3339", "2026-02-24T10:30:00Z"},
		{"RFC3339Nano", "2026-02-24T10:30:00.123456789Z"},
		{"RFC3339 with offset", "2026-02-24T10:30:00+05:00"},
		{"datetime with space", "2026-02-24 10:30:00"},
		{"datetime with fractional", "2026-02-24 10:30:00.123456"},
		{"datetime with offset", "2026-02-24 10:30:00.123-05:00"},
		{"datetime T no timezone", "2026-02-24T10:30:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamp
			require.NoError(t, ts.Scan(tt.input), "Scan(%q) error", tt.input)
			if ts.IsZero() {
				t.Errorf("Scan(%q) produced zero time", tt.input)
			}
		})
	}
}

func TestTimestamp_Scan_InvalidString(t *testing.T) {
	var ts Timestamp
	err := ts.Scan("not-a-timestamp")
	if err == nil {
		t.Error("expected error for invalid string, got nil")
	}
}

func TestTimestamp_Scan_UnsupportedType(t *testing.T) {
	var ts Timestamp
	err := ts.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestTimestamp_Scan_StringWithoutTimezoneIsUTC(t *testing.T) {
	var ts Timestamp
	require.NoError(t, ts.Scan("2026-02-24 10:30:00"), "Scan error")
	if ts.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", ts.Location())
	}

	var ts2 Timestamp
	require.NoError(t, ts2.Scan("2026-02-24T10:30:00"), "Scan error")
	if ts2.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", ts2.Location())
	}
}

func TestTimestamp_MarshalJSON_NonZero(t *testing.T) {
	ts := Timestamp{time.Date(2026, 2, 24, 10, 30, 0, 0, time.UTC)}
	data, err := json.Marshal(ts)
	require.NoError(t, err, "MarshalJSON error")
	want := `"2026-02-24T10:30:00Z"`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestTimestamp_MarshalJSON_Zero(t *testing.T) {
	var ts Timestamp
	data, err := json.Marshal(ts)
	require.NoError(t, err, "MarshalJSON error")
	want := `""`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestTimestamp_UnmarshalJSON_Valid(t *testing.T) {
	var ts Timestamp
	require.NoError(t, json.Unmarshal([]byte(`"2026-02-24T10:30:00Z"`), &ts), "UnmarshalJSON error")
	want := time.Date(2026, 2, 24, 10, 30, 0, 0, time.UTC)
	if !ts.Equal(want) {
		t.Errorf("got %v, want %v", ts.Time, want)
	}
}

func TestTimestamp_UnmarshalJSON_Empty(t *testing.T) {
	var ts Timestamp
	ts.Time = time.Now()
	require.NoError(t, json.Unmarshal([]byte(`""`), &ts), "UnmarshalJSON error")
	if !ts.IsZero() {
		t.Errorf("expected zero time for empty string, got %v", ts.Time)
	}
}

func TestTimestamp_UnmarshalJSON_Null(t *testing.T) {
	var ts Timestamp
	ts.Time = time.Now()
	require.NoError(t, json.Unmarshal([]byte("null"), &ts), "UnmarshalJSON error")
	if !ts.IsZero() {
		t.Errorf("expected zero time for null, got %v", ts.Time)
	}
}

func TestTimestamp_UnmarshalJSON_Invalid(t *testing.T) {
	var ts Timestamp
	err := json.Unmarshal([]byte(`"not-a-timestamp"`), &ts)
	if err == nil {
		t.Error("expected error for invalid timestamp, got nil")
	}
}

func TestTimestamp_JSONRoundTrip(t *testing.T) {
	original := Timestamp{time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(original)
	require.NoError(t, err, "Marshal error")

	var decoded Timestamp
	require.NoError(t, json.Unmarshal(data, &decoded), "Unmarshal error")

	if !original.Equal(decoded.Time) {
		t.Errorf("round-trip failed: got %v, want %v", decoded.Time, original.Time)
	}
}
