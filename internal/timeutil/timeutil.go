// Package timeutil provides shared time formatting helpers.
package timeutil

import "time"

// NowRFC3339 returns the current UTC time formatted as RFC 3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
