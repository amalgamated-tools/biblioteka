package exif

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var readyToken = []byte("{ready}\n")

const writeMetadataSuccessToken = "image files updated\n"

var exiftoolBinary = "exiftool"

var executeArg = "-execute"
var initArgs = []string{"-stay_open", "True", "-@", "-"}

var extractArgs = []string{"-ee3", "-U", "-api", "requestall=3", "-api", "largefilesupport", "-t"}
var closeArgs = []string{"-stay_open", "False", executeArg}
var readyTokenLen = len(readyToken)

// WaitTimeout specifies the duration to wait for exiftool to exit when closing before timing out
var WaitTimeout = time.Second

// ErrNotExist is a sentinel error for non existing file
var ErrNotExist = errors.New("file does not exist")

// ErrNotFile is a sentinel error that is returned when a folder is provided instead of a rerular file
var ErrNotFile = errors.New("can't extract metadata from folder")

// ErrBufferTooSmall is a sentinel error that is returned when the buffer used to store Exiftool's output is too small.
var ErrBufferTooSmall = errors.New("exiftool's buffer too small (see Buffer init option)")

// Exiftool is the exiftool utility wrapper
type Exiftool struct {
	lock                     sync.Mutex
	stdin                    io.WriteCloser
	stdMergedOut             io.Reader
	scanMergedOut            *bufio.Scanner
	bufferSet                bool
	buffer                   []byte
	bufferMaxSize            int
	extraInitArgs            []string
	cmd                      *exec.Cmd
	backupOriginal           bool
	clearFieldsBeforeWriting bool
}

// NewExiftool instanciates a new Exiftool with configuration functions. If anything went
// wrong, a non empty error will be returned.
func NewExiftool(opts ...func(*Exiftool) error) (*Exiftool, error) {
	e := Exiftool{}

	for _, opt := range opts {
		if err := opt(&e); err != nil {
			return nil, fmt.Errorf("error when configuring exiftool: %w", err)
		}
	}

	args := append([]string(nil), initArgs...)
	if len(e.extraInitArgs) > 0 {
		args = append(args, "-common_args")
		args = append(args, e.extraInitArgs...)
	}

	e.cmd = exec.Command(exiftoolBinary, args...)

	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error when piping stdout: %w", err)
	}

	stderr, err := e.cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("error when piping stderr: %w", err)
	}

	e.stdMergedOut = io.MultiReader(stdout, stderr)

	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		return nil, fmt.Errorf("error when piping stdin: %w", err)
	}

	e.scanMergedOut = bufio.NewScanner(e.stdMergedOut)
	if e.bufferSet {
		e.scanMergedOut.Buffer(e.buffer, e.bufferMaxSize)
	}
	e.scanMergedOut.Split(splitReadyToken)

	if err = e.cmd.Start(); err != nil {
		return nil, fmt.Errorf("error when executing command: %w", err)
	}

	return &e, nil
}

// Close closes exiftool. If anything went wrong, a non empty error will be returned
func (e *Exiftool) Close() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, v := range closeArgs {
		_, err := fmt.Fprintln(e.stdin, v)
		if err != nil {
			return err
		}
	}

	var errs []error
	if err := e.stdin.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error while closing stdin: %w", err))
	}

	ch := make(chan struct{})
	go func() {
		if e.cmd != nil {
			if err := e.cmd.Wait(); err != nil {
				errs = append(errs, fmt.Errorf("error while waiting for exiftool to exit: %w", err))
			}
		}
		ch <- struct{}{}
		close(ch)
	}()

	// Wait for wait to finish or timeout
	select {
	case <-ch:
	case <-time.After(WaitTimeout):
		errs = append(errs, errors.New("Timed out waiting for exiftool to exit"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("error while closing exiftool: %v", errs)
	}

	return nil
}

func (e *Exiftool) ExtractMetadataFromFile(file string) (*ExifToolOutput, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	s, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExist
		}
		return nil, err
	}

	if s.IsDir() {
		return nil, ErrNotFile
	}

	fileFormat := filepath.Ext(file)
	if fileFormat == "" {
		return nil, fmt.Errorf("can't extract metadata from file without extension")
	}

	for _, curA := range extractArgs {
		if _, err := fmt.Fprintln(e.stdin, curA); err != nil {
			return nil, err
		}
	}

	if _, err := fmt.Fprintln(e.stdin, file); err != nil {
		return nil, err
	}

	if _, err := fmt.Fprintln(e.stdin, executeArg); err != nil {
		return nil, err
	}

	scanOk := e.scanMergedOut.Scan()
	scanErr := e.scanMergedOut.Err()
	if scanErr != nil {
		if scanErr == bufio.ErrTooLong {
			return nil, ErrBufferTooSmall
		}
		return nil, fmt.Errorf("error while reading stdMergedOut: %w", scanErr)
	}
	if !scanOk {
		return nil, fmt.Errorf("error while reading stdMergedOut: EOF")
	}
	debugFile := fmt.Sprintf("debug_%s.tsv", pathpkg.Base(file))
	os.WriteFile(debugFile, []byte(e.scanMergedOut.Text()), 0644)

	return ParseTSV(e.scanMergedOut.Text(), fileFormat)
}

// WriteMetadata writes the given metadata for each file.
// Any errors will be saved to FileMetadata.Err
// Note: If you're reusing an existing FileMetadata instance,
//
//	you should nil the Err before passing it to WriteMetadata
func (e *Exiftool) WriteMetadata(fileMetadata []FileMetadata) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for i, md := range fileMetadata {
		fileMetadata[i].Err = nil
		if _, err := os.Stat(md.File); err != nil {
			if os.IsNotExist(err) {
				fileMetadata[i].Err = ErrNotExist
				continue
			}
			fileMetadata[i].Err = err
			continue
		}

		if !e.backupOriginal {
			if _, err := fmt.Fprintln(e.stdin, "-overwrite_original"); err != nil {
				fileMetadata[i].Err = err
				continue
			}
		}

		if e.clearFieldsBeforeWriting {
			if _, err := fmt.Fprintln(e.stdin, "-All="); err != nil {
				fileMetadata[i].Err = err
				continue
			}
		}

		for k, v := range md.Fields {
			switch v.(type) {
			case nil:
				if _, err := fmt.Fprintln(e.stdin, "-"+k+"="); err != nil {
					fileMetadata[i].Err = err
					continue
				}
			default:
				strTab, err := md.GetStrings(k)
				if err != nil {
					fileMetadata[i].Err = err
					continue
				}
				for _, str := range strTab {
					// TODO: support writing an empty string via '^='
					if _, err := fmt.Fprintln(e.stdin, "-"+k+"="+str); err != nil {
						fileMetadata[i].Err = err
						continue
					}
				}
			}
		}

		if _, err := fmt.Fprintln(e.stdin, md.File); err != nil {
			fileMetadata[i].Err = err
			continue
		}
		if _, err := fmt.Fprintln(e.stdin, executeArg); err != nil {
			fileMetadata[i].Err = err
			continue
		}

		scanOk := e.scanMergedOut.Scan()
		scanErr := e.scanMergedOut.Err()
		if scanErr != nil {
			if scanErr == bufio.ErrTooLong {
				fileMetadata[i].Err = ErrBufferTooSmall
				continue
			}
			fileMetadata[i].Err = fmt.Errorf("error while reading stdMergedOut: %w", e.scanMergedOut.Err())
			continue
		}
		if !scanOk {
			fileMetadata[i].Err = fmt.Errorf("error while reading stdMergedOut: EOF")
			continue
		}

		if err := handleWriteMetadataResponse(e.scanMergedOut.Text()); err != nil {
			fileMetadata[i].Err = fmt.Errorf("Error writing metadata: %w", err)
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

const (
	defaultString = ""
	defaultFloat  = float64(0)
	defaultInt    = int64(0)
)

// ErrKeyNotFound is a sentinel error used when a queried key does not exist
var ErrKeyNotFound = errors.New("key not found")

// FileMetadata is a structure that represents an exiftool extraction. File contains the
// filename that had to be extracted. If anything went wrong, Err will not be nil. Fields
// stores extracted fields.
type FileMetadata struct {
	File   string
	Fields map[string]interface{}
	Output ExifToolOutput
	Err    error
}

// GetString returns a field value as string and an error if one occurred.
// KeyNotFoundError will be returned if the key can't be found
func (fm FileMetadata) GetString(k string) (string, error) {
	v, found := fm.Fields[k]
	if !found || v == nil {
		return defaultString, ErrKeyNotFound
	}

	return toString(v), nil
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

// GetFloat returns a field value as float64 and an error if one occurred.
// KeyNotFoundError will be returned if the key can't be found.
func (fm FileMetadata) GetFloat(k string) (float64, error) {
	v, found := fm.Fields[k]
	if !found || v == nil {
		return defaultFloat, ErrKeyNotFound
	}

	switch v := v.(type) {
	case string:
		return toFloatFallback(v)
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		str := fmt.Sprintf("%v", v)
		return toFloatFallback(str)
	}
}

func toFloatFallback(str string) (float64, error) {
	f, err := strconv.ParseFloat(str, -1)
	if err != nil {
		return defaultFloat, fmt.Errorf("float64 parsing error (%v): %w", str, err)
	}

	return f, nil
}

// GetInt returns a field value as int64 and an error if one occurred.
// KeyNotFoundError will be returned if the key can't be found, ParseError if
// a parsing error occurs.
func (fm FileMetadata) GetInt(k string) (int64, error) {
	v, found := fm.Fields[k]
	if !found || v == nil {
		return defaultInt, ErrKeyNotFound
	}

	switch v := v.(type) {
	case string:
		return toIntFallback(v)
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		str := fmt.Sprintf("%v", v)
		return toIntFallback(str)
	}
}

func toIntFallback(str string) (int64, error) {
	f, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultInt, fmt.Errorf("int64 parsing error (%v): %w", str, err)
	}

	return f, nil
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

// SetInt sets a int value for a specific field
func (fm FileMetadata) SetInt(k string, v int64) {
	fm.set(k, v)
}

// SetFloat sets a float value for a specific field
func (fm FileMetadata) SetFloat(k string, v float64) {
	fm.set(k, v)
}

// SetStrings sets a []String value for a specific field
func (fm FileMetadata) SetStrings(k string, v []string) {
	t := make([]interface{}, len(v))
	for i, c := range v {
		t[i] = c
	}
	fm.set(k, t)
}

// Clear removes value for a specific metadata field
func (fm FileMetadata) Clear(k string) {
	fm.set(k, nil)
}

// ClearAll removes all medatadata
func (fm FileMetadata) ClearAll() {
	for k := range fm.Fields {
		fm.set(k, nil)
	}
}

// EmptyFileMetadata creates an empty FileMetadata struct
func EmptyFileMetadata() FileMetadata {
	return FileMetadata{
		Fields: make(map[string]interface{}),
	}
}
