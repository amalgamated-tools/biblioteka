package jobs

import (
	"context"
	"encoding/json"
	"fmt"
)

// JobScanPath is the registered name for the path-scanning job.
const JobScanPath = "scan:path"

// ScanPathPayload is the JSON payload for the scan:path job.
type ScanPathPayload struct {
	Path        string `json:"path"`
	LibraryID   string `json:"library_id,omitempty"`
	LibraryRoot string `json:"library_root,omitempty"`
}

// NewScanPathHandler returns a worker.Func that walks the given path and
// enqueues a process:file job for every EPUB, MOBI, PDF, or AZW3 it finds.
func NewScanPathHandler(enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p ScanPathPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal scan path payload: %w", err)
		}

		return ScanDirectory(ctx, enqueuer, p)
	}
}
