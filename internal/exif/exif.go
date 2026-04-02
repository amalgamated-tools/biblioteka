package exif

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var readyToken = []byte("{ready}\n")

const writeMetadataSuccessToken = "image files updated\n"

var exiftoolBinary = "exiftool"

var (
	executeArg = "-execute"
	initArgs   = []string{"-stay_open", "True", "-@", "-"}
)

var (
	extractArgs   = []string{"-ee3", "-U", "-api", "requestall=3", "-api", "largefilesupport", "-t"}
	closeArgs     = []string{"-stay_open", "False", executeArg}
	readyTokenLen = len(readyToken)
)

// WaitTimeout specifies the duration to wait for exiftool to exit when closing before timing out
var WaitTimeout = time.Second

// ErrNotExist is a sentinel error for non existing file
var ErrNotExist = errors.New("file does not exist")

// ErrNotFile is a sentinel error that is returned when a folder is provided instead of a regular file
var ErrNotFile = errors.New("can't extract metadata from folder")

// ErrBufferTooSmall is a sentinel error that is returned when the buffer used to store Exiftool's output is too small.
var ErrBufferTooSmall = errors.New("exiftool's buffer too small (see Buffer init option)")

// ErrNoCover is returned when a book file has no embedded cover image.
// This is a normal condition for many files and should not be treated as a failure.
var ErrNoCover = errors.New("no cover found")

// ErrDead is returned when a write failure has corrupted the exiftool stdin protocol.
// Once this error is returned, the instance cannot be reused.
var ErrDead = errors.New("exiftool instance is dead due to a previous write failure")

// Exiftool is the exiftool utility wrapper
type Exiftool struct {
	lock                     sync.Mutex
	dead                     bool
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
func NewExiftool(ctx context.Context, opts ...func(*Exiftool) error) (*Exiftool, error) {
	e := Exiftool{}

	for _, opt := range opts {
		if err := opt(&e); err != nil {
			slog.ErrorContext(ctx, "error when configuring exiftool", slog.Any(otelkeys.Error, err))
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
		slog.ErrorContext(ctx, "error when piping stdout", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("error when piping stdout: %w", err)
	}

	stderr, err := e.cmd.StderrPipe()
	if err != nil {
		slog.ErrorContext(ctx, "error when piping stderr", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("error when piping stderr: %w", err)
	}

	// Drain stderr in the background so it cannot block the child process.
	// io.MultiReader would read stdout to EOF before stderr, but in -stay_open
	// mode stdout never reaches EOF until the process exits, so stderr would
	// never be drained and could deadlock the child if the pipe buffer fills.
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()

	e.stdMergedOut = stdout

	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		slog.ErrorContext(ctx, "error when piping stdin", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("error when piping stdin: %w", err)
	}

	e.scanMergedOut = bufio.NewScanner(e.stdMergedOut)
	if e.bufferSet {
		e.scanMergedOut.Buffer(e.buffer, e.bufferMaxSize)
	}
	e.scanMergedOut.Split(splitReadyToken)

	if err = e.cmd.Start(); err != nil {
		slog.ErrorContext(ctx, "error when starting exiftool process", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("error when starting exiftool process: %w", err)
	}

	return &e, nil
}

// Close closes exiftool. If anything went wrong, a non empty error will be returned
func (e *Exiftool) Close(ctx context.Context) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, v := range closeArgs {
		if _, err := fmt.Fprintln(e.stdin, v); err != nil {
			slog.ErrorContext(ctx, "error while sending close command to exiftool", slog.Any(otelkeys.Error, err))
			_ = e.stdin.Close()
			_ = e.cmd.Wait()
			return fmt.Errorf("error while sending close command to exiftool: %w", err)
		}
	}

	var errs []error
	if err := e.stdin.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error while closing stdin: %w", err))
	}

	ch := make(chan error, 1)
	go func() {
		if e.cmd != nil {
			ch <- e.cmd.Wait()
		} else {
			ch <- nil
		}
	}()

	// Wait for wait to finish or timeout
	select {
	case waitErr := <-ch:
		if waitErr != nil {
			errs = append(errs, fmt.Errorf("error while waiting for exiftool to exit: %w", waitErr))
		}
	case <-time.After(WaitTimeout):
		errs = append(errs, errors.New("timed out waiting for exiftool to exit"))

		// Ensure the exiftool process does not leak by killing it and waiting for exit.
		if e.cmd != nil && e.cmd.Process != nil {
			if killErr := e.cmd.Process.Kill(); killErr != nil {
				errs = append(errs, fmt.Errorf("error while killing exiftool process after timeout: %w", killErr))
			} else {
				if waitErr := <-ch; waitErr != nil {
					errs = append(errs, fmt.Errorf("error while waiting for exiftool to exit after kill: %w", waitErr))
				}
			}
		}
	}

	if len(errs) > 0 {
		joinedErr := errors.Join(errs...)
		slog.ErrorContext(ctx, "errors while closing exiftool", slog.Any(otelkeys.Error, joinedErr))
		return fmt.Errorf("error while closing exiftool: %w", joinedErr)
	}

	return nil
}

// markDead poisons the instance after a partial stdin write so that future
// calls return ErrDead instead of silently corrupting the exiftool protocol.
// Must be called while e.lock is held.
func (e *Exiftool) markDead() {
	e.dead = true
	_ = e.stdin.Close()
	if e.cmd != nil {
		_ = e.cmd.Wait()
	}
}

func (e *Exiftool) ExtractMetadataFromFile(ctx context.Context, file string) (*ExifToolOutput, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.dead {
		return nil, ErrDead
	}

	s, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			slog.WarnContext(ctx, "file does not exist", slog.String(otelkeys.Path, file))
			return nil, ErrNotExist
		}
		return nil, err
	}

	if s.IsDir() {
		return nil, ErrNotFile
	}

	fileFormat := filepath.Ext(file)
	if fileFormat == "" {
		slog.WarnContext(ctx, "file has no extension, can't determine format", slog.String(otelkeys.Path, file))
		return nil, fmt.Errorf("can't extract metadata from file without extension")
	}

	for _, curA := range extractArgs {
		if _, err := fmt.Fprintln(e.stdin, curA); err != nil {
			slog.WarnContext(ctx, "failed to write extract argument", slog.Any(otelkeys.Error, err))
			e.markDead()
			return nil, err
		}
	}

	if _, err := fmt.Fprintln(e.stdin, file); err != nil {
		slog.WarnContext(
			ctx,
			"failed to write file path",
			slog.String(otelkeys.Path, file),
			slog.Any(otelkeys.Error, err),
		)
		e.markDead()
		return nil, err
	}

	if _, err := fmt.Fprintln(e.stdin, executeArg); err != nil {
		slog.WarnContext(ctx, "failed to write execute argument", slog.Any(otelkeys.Error, err))
		e.markDead()
		return nil, err
	}

	scanOk := e.scanMergedOut.Scan()
	scanErr := e.scanMergedOut.Err()
	if scanErr != nil {
		slog.WarnContext(
			ctx,
			"error while reading exiftool output",
			slog.String(otelkeys.Error, scanErr.Error()),
		)
		// Any scanner error leaves the instance in a broken state.
		e.markDead()
		if scanErr == bufio.ErrTooLong {
			return nil, ErrBufferTooSmall
		}
		return nil, fmt.Errorf("error while reading stdMergedOut: %w", scanErr)
	}
	if !scanOk {
		slog.WarnContext(
			ctx,
			"unexpected EOF while reading exiftool output",
			slog.String(otelkeys.Error, "EOF"),
		)
		e.markDead()
		return nil, fmt.Errorf("error while reading stdMergedOut: EOF")
	}

	return ParseTSV(ctx, e.scanMergedOut.Text(), fileFormat)
}
