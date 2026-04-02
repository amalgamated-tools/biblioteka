package exif

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// WriteMetadata writes the given metadata for each file.
// Any errors will be saved to FileMetadata.Err
// Note: If you're reusing an existing FileMetadata instance,
//
//	you should nil the Err before passing it to WriteMetadata
func (e *Exiftool) WriteMetadata(ctx context.Context, fileMetadata []FileMetadata) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.dead {
		for i := range fileMetadata {
			fileMetadata[i].Err = ErrDead
		}
		return
	}

	for i, md := range fileMetadata {
		fileMetadata[i].Err = nil
		if _, err := os.Stat(md.File); err != nil {
			slog.WarnContext(
				ctx,
				"file does not exist",
				slog.String(otelkeys.File, md.File),
				slog.Any(otelkeys.Error, err),
			)
			if os.IsNotExist(err) {
				fileMetadata[i].Err = ErrNotExist
				continue
			}
			fileMetadata[i].Err = err
			continue
		}

		if !e.backupOriginal {
			if _, err := fmt.Fprintln(e.stdin, "-overwrite_original"); err != nil {
				slog.WarnContext(ctx, "failed to set overwrite_original flag for exiftool", slog.Any(otelkeys.Error, err))
				fileMetadata[i].Err = err
				e.markDead()
				return
			}
		}

		if e.clearFieldsBeforeWriting {
			if _, err := fmt.Fprintln(e.stdin, "-All="); err != nil {
				slog.WarnContext(ctx, "failed to set All= flag for exiftool", slog.Any(otelkeys.Error, err))
				fileMetadata[i].Err = err
				e.markDead()
				return
			}
		}

		for k, v := range md.Fields {
			switch v.(type) {
			case nil:
				if _, err := fmt.Fprintln(e.stdin, "-"+k+"="); err != nil {
					slog.WarnContext(
						ctx,
						"failed to write empty value for field",
						slog.String(otelkeys.Field, k),
						slog.Any(otelkeys.Error, err),
					)
					fileMetadata[i].Err = err
					e.markDead()
					return
				}
			default:
				strTab, err := md.GetStrings(k)
				if err != nil {
					slog.WarnContext(
						ctx,
						"failed to convert field value to string slice",
						slog.String(otelkeys.Field, k),
						slog.Any(otelkeys.Error, err),
					)
					fileMetadata[i].Err = err
					// Args may have already been written to stdin for this file
					// (e.g., -overwrite_original), so the protocol is corrupted.
					e.markDead()
					return
				}
				for _, str := range strTab {
					// TODO: support writing an empty string via '^='
					if _, err := fmt.Fprintln(e.stdin, "-"+k+"="+str); err != nil {
						slog.WarnContext(
							ctx,
							"failed to write field value",
							slog.String(otelkeys.Field, k),
							slog.Any(otelkeys.Error, err),
						)
						fileMetadata[i].Err = err
						e.markDead()
						return
					}
				}
			}
		}

		if _, err := fmt.Fprintln(e.stdin, md.File); err != nil {
			slog.WarnContext(
				ctx,
				"failed to write file path",
				slog.String(otelkeys.File, md.File),
				slog.Any(otelkeys.Error, err),
			)
			fileMetadata[i].Err = err
			e.markDead()
			return
		}
		if _, err := fmt.Fprintln(e.stdin, executeArg); err != nil {
			slog.WarnContext(ctx, "failed to write execute argument", slog.Any(otelkeys.Error, err))
			fileMetadata[i].Err = err
			e.markDead()
			return
		}

		scanOk := e.scanMergedOut.Scan()
		scanErr := e.scanMergedOut.Err()
		if scanErr != nil {
			slog.WarnContext(
				ctx,
				"error while reading exiftool output",
				slog.String(otelkeys.Error, scanErr.Error()),
			)
			if scanErr == bufio.ErrTooLong {
				fileMetadata[i].Err = ErrBufferTooSmall
				continue
			}
			fileMetadata[i].Err = fmt.Errorf("error while reading stdMergedOut: %w", e.scanMergedOut.Err())
			continue
		}
		if !scanOk {
			slog.WarnContext(
				ctx,
				"unexpected EOF while reading exiftool output",
				slog.String(otelkeys.Error, "EOF"),
			)
			fileMetadata[i].Err = fmt.Errorf("error while reading stdMergedOut: EOF")
			continue
		}

		if err := handleWriteMetadataResponse(e.scanMergedOut.Text()); err != nil {
			slog.WarnContext(
				ctx,
				"exiftool reported error while writing metadata",
				slog.Any(otelkeys.Error, err),
			)
			fileMetadata[i].Err = fmt.Errorf("error writing metadata: %w", err)
			continue
		}
	}
}

func splitReadyToken(data []byte, atEOF bool) (int, []byte, error) {
	idx := bytes.Index(data, readyToken)
	if idx == -1 {
		if atEOF && len(data) > 0 {
			return 0, data, fmt.Errorf("no final token found")
		}

		return 0, nil, nil
	}

	return idx + readyTokenLen, data[:idx], nil
}

func handleWriteMetadataResponse(resp string) error {
	if strings.HasSuffix(resp, writeMetadataSuccessToken) {
		return nil
	}
	return errors.New(strings.TrimSpace(resp))
}

// ErrKeyNotFound is a sentinel error used when a queried key does not exist
var ErrKeyNotFound = errors.New("key not found")

// FileMetadata is a structure that represents an exiftool extraction. File contains the
// filename that had to be extracted. If anything went wrong, Err will not be nil. Fields
// stores extracted fields.
type FileMetadata struct {
	File   string
	Fields map[string]interface{}
	Err    error
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetStrings returns a field value as []string and an error if one occurred.
// KeyNotFoundError will be returned if the key can't be found.
func (fm FileMetadata) GetStrings(k string) ([]string, error) {
	v, found := fm.Fields[k]
	if !found || v == nil {
		return []string{}, ErrKeyNotFound
	}

	switch v := v.(type) {
	case []interface{}:
		is := v
		res := make([]string, len(is))

		for i, v2 := range is {
			res[i] = toString(v2)
		}

		return res, nil
	default:
		return []string{toString(v)}, nil
	}
}

func (fm FileMetadata) set(k string, v interface{}) {
	fm.Fields[k] = v
}

// SetString sets a string value for a specific field
func (fm FileMetadata) SetString(k string, v string) {
	fm.set(k, v)
}

// EmptyFileMetadata creates an empty FileMetadata struct
func EmptyFileMetadata() FileMetadata {
	return FileMetadata{
		Fields: make(map[string]interface{}),
	}
}
