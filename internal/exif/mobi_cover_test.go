package exif

// mobi_cover_test.go tests GetMobiCover.
//
// IMPORTANT: This file must NOT import internal/testutils. The testutils
// package imports internal/exif (via testutils/pdf.go → exif) which would
// create an import cycle. The minimal MOBI builder below is a subset of the
// one in testutils/mobi.go kept here specifically to avoid that cycle.

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// tinyJPEGForMobiCoverTest returns a 1×1 white JPEG suitable as a cover image.
func tinyJPEGForMobiCoverTest() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 1}); err != nil {
		panic("mobi_cover_test: failed to encode tiny JPEG: " + err.Error())
	}
	return buf.Bytes()
}

// makeMiniMOBI writes a minimal PalmDB/MOBI file to path. When coverData is
// non-nil the cover is embedded as record 2 and the EXTH CoverOffset field
// is set accordingly.
func makeMiniMOBI(t *testing.T, path, title string, coverData []byte) {
	t.Helper()
	hasCover := len(coverData) > 0
	numRecords := uint16(2)
	if hasCover {
		numRecords = 3
	}

	const mobiHeaderLen = 232
	var mobi [mobiHeaderLen]byte
	copy(mobi[0:4], "MOBI")
	binary.BigEndian.PutUint32(mobi[4:8], mobiHeaderLen)
	binary.BigEndian.PutUint32(mobi[8:12], 2)      // MobiType
	binary.BigEndian.PutUint32(mobi[12:16], 65001) // UTF-8

	// Build EXTH block only when we have a cover.
	var exthBlock []byte
	if hasCover {
		binary.BigEndian.PutUint32(mobi[112:116], 0x40) // ExthFlags
		// Single EXTH record: CoverOffset (type 201) = 0 (first image record).
		rec := make([]byte, 12) // type(4) + length(4) + value(4)
		binary.BigEndian.PutUint32(rec[0:4], 201)
		binary.BigEndian.PutUint32(rec[4:8], 12)
		binary.BigEndian.PutUint32(rec[8:12], 0)

		totalSize := 12 + 12 // EXTH header (12) + one record (12) = 24; already 4-byte aligned
		exthBlock = make([]byte, totalSize)
		copy(exthBlock[0:4], "EXTH")
		binary.BigEndian.PutUint32(exthBlock[4:8], uint32(totalSize))
		binary.BigEndian.PutUint32(exthBlock[8:12], 1) // 1 record
		copy(exthBlock[12:], rec)

		binary.BigEndian.PutUint32(mobi[92:96], 2) // FirstImageIndex = record 2
	} else {
		binary.BigEndian.PutUint32(mobi[92:96], math.MaxUint32)
	}
	binary.BigEndian.PutUint32(mobi[64:68], uint32(numRecords)) // FirstNonBookIndex
	for _, off := range []int{24, 28, 32, 36, 40, 44, 48, 52, 56, 60} {
		binary.BigEndian.PutUint32(mobi[off:off+4], math.MaxUint32)
	}
	binary.BigEndian.PutUint32(mobi[148:152], math.MaxUint32) // DrmOffset
	binary.BigEndian.PutUint32(mobi[152:156], math.MaxUint32) // DrmCount
	binary.BigEndian.PutUint16(mobi[176:178], 1)
	binary.BigEndian.PutUint16(mobi[178:180], 1)
	binary.BigEndian.PutUint32(mobi[184:188], math.MaxUint32)
	binary.BigEndian.PutUint32(mobi[188:192], 1)
	binary.BigEndian.PutUint32(mobi[192:196], math.MaxUint32)
	binary.BigEndian.PutUint32(mobi[196:200], 1)
	binary.BigEndian.PutUint32(mobi[228:232], math.MaxUint32)

	fullNameOffset := uint32(16 + mobiHeaderLen + len(exthBlock))
	binary.BigEndian.PutUint32(mobi[68:72], fullNameOffset)
	binary.BigEndian.PutUint32(mobi[72:76], uint32(len(title)))

	var pdh [16]byte
	textContent := []byte("Hello, world.")
	binary.BigEndian.PutUint16(pdh[0:2], 1)
	binary.BigEndian.PutUint32(pdh[4:8], uint32(len(textContent)))
	binary.BigEndian.PutUint16(pdh[8:10], 1)
	binary.BigEndian.PutUint16(pdh[10:12], 4096)

	record0 := append(append(append(pdh[:], mobi[:]...), exthBlock...), []byte(title)...)
	records := [][]byte{record0, textContent}
	if hasCover {
		records = append(records, coverData)
	}

	// PalmDB header.
	var pdb [78]byte
	dbName := title
	if len(dbName) > 31 {
		dbName = dbName[:31]
	}
	copy(pdb[0:32], dbName)
	copy(pdb[60:64], "BOOK")
	copy(pdb[64:68], "MOBI")
	binary.BigEndian.PutUint16(pdb[76:78], numRecords)

	offsetTable := make([]byte, int(numRecords)*8)
	dataStart := 78 + len(offsetTable) + 2
	cur := dataStart
	for i := range int(numRecords) {
		base := i * 8
		binary.BigEndian.PutUint32(offsetTable[base:base+4], uint32(cur))
		binary.BigEndian.PutUint16(offsetTable[base+6:base+8], uint16(i))
		cur += len(records[i])
	}

	var file []byte
	file = append(file, pdb[:]...)
	file = append(file, offsetTable...)
	file = append(file, 0, 0)
	for _, rec := range records {
		file = append(file, rec...)
	}

	require.NoError(t, os.WriteFile(path, file, 0o644))
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestGetMobiCover_WithCover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "withcover.mobi")
	makeMiniMOBI(t, path, "Cover Book", tinyJPEGForMobiCoverTest())

	dataURL, err := GetMobiCover(t.Context(), path)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(dataURL, "data:image/jpeg;base64,"),
		"expected JPEG data URL, got %q", dataURL)
	require.NotEmpty(t, strings.TrimPrefix(dataURL, "data:image/jpeg;base64,"),
		"base64 payload must not be empty")
}

func TestGetMobiCover_AZW3_WithCover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "withcover.azw3")
	makeMiniMOBI(t, path, "AZW3 Book", tinyJPEGForMobiCoverTest())

	dataURL, err := GetMobiCover(t.Context(), path)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(dataURL, "data:image/jpeg;base64,"),
		"AZW3 file: expected JPEG data URL, got %q", dataURL)
}

func TestGetMobiCover_NoCover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nocover.mobi")
	makeMiniMOBI(t, path, "No Cover Book", nil)

	_, err := GetMobiCover(t.Context(), path)
	require.ErrorIs(t, err, ErrNoCover,
		"MOBI file with no cover should return ErrNoCover")
}

func TestGetMobiCover_FileNotFound(t *testing.T) {
	_, err := GetMobiCover(t.Context(), "/nonexistent/path/book.mobi")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoCover,
		"missing file should not return ErrNoCover")
}
