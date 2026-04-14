package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/storage"
	"github.com/stretchr/testify/require"
)

func newLocalStorage() *storage.LocalStorage {
	return storage.NewLocalStorage()
}

func TestLocalStorage_Open(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello world"), 0o644))

	ls := newLocalStorage()
	rc, err := ls.Open(context.Background(), path)
	require.NoError(t, err)
	defer rc.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rc)
	require.NoError(t, err)
	require.Equal(t, "hello world", buf.String())
}

func TestLocalStorage_Open_NotExist(t *testing.T) {
	ls := newLocalStorage()
	_, err := ls.Open(context.Background(), "/nonexistent/path/file.txt")
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Stat_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stat.txt")
	content := []byte("stat me")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	before := time.Now().Truncate(time.Second)

	ls := newLocalStorage()
	fi, err := ls.Stat(context.Background(), path)
	require.NoError(t, err)

	require.Equal(t, "stat.txt", fi.Name)
	require.Equal(t, int64(len(content)), fi.Size)
	require.False(t, fi.IsDir)
	require.False(t, fi.ModTime.Before(before))
}

func TestLocalStorage_Stat_Dir(t *testing.T) {
	dir := t.TempDir()

	ls := newLocalStorage()
	fi, err := ls.Stat(context.Background(), dir)
	require.NoError(t, err)

	require.True(t, fi.IsDir)
	require.Equal(t, filepath.Base(dir), fi.Name)
}

func TestLocalStorage_Stat_NotExist(t *testing.T) {
	ls := newLocalStorage()
	_, err := ls.Stat(context.Background(), "/nonexistent/path/file.txt")
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	ls := newLocalStorage()
	err := ls.Write(context.Background(), path, strings.NewReader("written content"))
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "written content", string(got))
}

func TestLocalStorage_Write_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	ls := newLocalStorage()
	err := ls.Write(context.Background(), path, strings.NewReader("new content"))
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new content", string(got))
}

func TestLocalStorage_Write_MissingDir(t *testing.T) {
	ls := newLocalStorage()
	err := ls.Write(context.Background(), "/nonexistent/dir/file.txt", strings.NewReader("x"))
	require.Error(t, err)
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	require.NoError(t, os.WriteFile(path, []byte("bye"), 0o644))

	ls := newLocalStorage()
	require.NoError(t, ls.Delete(context.Background(), path))

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Delete_NotExist(t *testing.T) {
	ls := newLocalStorage()
	err := ls.Delete(context.Background(), "/nonexistent/path/file.txt")
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLocalStorage_List(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bb"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	ls := newLocalStorage()
	entries, err := ls.List(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	names := make(map[string]storage.FileInfo, len(entries))
	for _, e := range entries {
		names[e.Name] = e
	}

	a := names["a.txt"]
	require.False(t, a.IsDir)
	require.Equal(t, int64(1), a.Size)

	b := names["b.txt"]
	require.False(t, b.IsDir)
	require.Equal(t, int64(2), b.Size)

	sub := names["sub"]
	require.True(t, sub.IsDir)
}

func TestLocalStorage_List_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("x"), 0o644))

	ls := newLocalStorage()
	entries, err := ls.List(context.Background(), dir)
	require.NoError(t, err)

	// Only "sub" should appear; "nested.txt" is deeper.
	require.Len(t, entries, 1)
	require.Equal(t, "sub", entries[0].Name)
	require.True(t, entries[0].IsDir)
}

func TestLocalStorage_List_NotExist(t *testing.T) {
	ls := newLocalStorage()
	_, err := ls.List(context.Background(), "/nonexistent/dir")
	require.Error(t, err)
}

func TestLocalStorage_List_Empty(t *testing.T) {
	dir := t.TempDir()

	ls := newLocalStorage()
	entries, err := ls.List(context.Background(), dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}
