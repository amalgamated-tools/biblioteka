package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockSettingGetter returns preset values for known keys.
type mockSettingGetter struct {
	settings map[string]string
	err      error
}

func (m *mockSettingGetter) GetSetting(_ context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	v, ok := m.settings[key]
	if !ok {
		return "", sql.ErrNoRows
	}
	return v, nil
}

func TestScanWatchFolderHandler_NotConfigured(t *testing.T) {
	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))
	require.Len(t, enq.jobs, 0, "no jobs should be enqueued when not configured")
}

func TestScanWatchFolderHandler_EmptyPath(t *testing.T) {
	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{
		settingWatchFolderPath:      "",
		settingWatchFolderLibraryID: "lib1",
	}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))
	require.Len(t, enq.jobs, 0, "no jobs should be enqueued when path is empty")
}

func TestScanWatchFolderHandler_EmptyLibraryID(t *testing.T) {
	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{
		settingWatchFolderPath:      "/some/path",
		settingWatchFolderLibraryID: "",
	}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))
	require.Len(t, enq.jobs, 0, "no jobs should be enqueued when library ID is empty")
}

func TestScanWatchFolderHandler_ScansDirectory(t *testing.T) {
	// Create a temp directory with a supported file.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.epub"), []byte("fake epub"), 0o644))

	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{
		settingWatchFolderPath:      dir,
		settingWatchFolderLibraryID: "lib-123",
	}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))

	require.Len(t, enq.jobs, 1, "should enqueue one process:file job")
	require.Equal(t, JobProcessFile, enq.jobs[0].Name)

	var p ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &p))
	require.Equal(t, "lib-123", p.LibraryID)
	require.Equal(t, "test.epub", p.FileName)
	require.Equal(t, "epub", p.FileType)
}

func TestScanWatchFolderHandler_IgnoresUnsupportedFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.doc"), []byte("notes"), 0o644))

	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{
		settingWatchFolderPath:      dir,
		settingWatchFolderLibraryID: "lib-123",
	}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))

	require.Len(t, enq.jobs, 0, "no jobs for unsupported file types")
}

func TestScanWatchFolderHandler_SettingsError(t *testing.T) {
	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{err: errors.New("db connection error")}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.Error(t, handler(t.Context(), nil))
}

func TestScanWatchFolderHandler_MultipleSupportedFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book1.epub"), []byte("epub"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book2.pdf"), []byte("pdf"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book3.mobi"), []byte("mobi"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book4.azw3"), []byte("azw3"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("txt"), 0o644))

	enq := &genericMockEnqueuer{}
	settings := &mockSettingGetter{settings: map[string]string{
		settingWatchFolderPath:      dir,
		settingWatchFolderLibraryID: "lib-456",
	}}

	handler := NewScanWatchFolderHandler(settings, enq)
	require.NoError(t, handler(t.Context(), nil))

	require.Len(t, enq.jobs, 4, "should enqueue 4 process:file jobs for supported types")
	for _, j := range enq.jobs {
		require.Equal(t, JobProcessFile, j.Name)
	}
}
