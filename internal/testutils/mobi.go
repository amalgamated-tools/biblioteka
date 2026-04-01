package testutils

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"testing"
)

// MOBIOptions controls optional metadata and cover image in a synthetic MOBI file.
type MOBIOptions struct {
	ISBN           string
	ASIN           string
	Publisher      string
	Language       string
	CoverImageData []byte // Raw image bytes (e.g., TinyJPEG()); must be decodable by image.Decode.
}

// MakeTestMOBI creates a minimal valid MOBI file at path with the given metadata.
// The file is readable by both exiftool and the sblinch/mobi library.
func MakeTestMOBI(t *testing.T, path, title, author string, opts MOBIOptions) {
	t.Helper()
	data := buildMOBI(t, title, author, opts)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write mobi: %v", err)
	}
}

// MakeTestAZW3 creates a minimal valid AZW3 file at path. AZW3 uses the same
// PalmDB/MOBI binary format as MOBI; only the file extension differs.
func MakeTestAZW3(t *testing.T, path, title, author string, opts MOBIOptions) {
	t.Helper()
	data := buildMOBI(t, title, author, opts)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write azw3: %v", err)
	}
}

// TinyJPEG returns a minimal valid 1x1 pixel JPEG image that can be decoded
// by Go's image.Decode (required by GetMobiCover).
func TinyJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 1}); err != nil {
		panic("failed to encode tiny JPEG: " + err.Error())
	}
	return buf.Bytes()
}

// EXTH record type constants.
const (
	exthTitle       = 99
	exthAuthor      = 100
	exthPublisher   = 101
	exthISBN        = 104
	exthASIN        = 113
	exthCoverOffset = 201
	exthLanguage    = 524
)

// buildMOBI constructs a minimal MOBI binary in memory.
func buildMOBI(t *testing.T, title, author string, opts MOBIOptions) []byte {
	t.Helper()

	// Determine record count: record 0 (header), record 1 (text), optionally record 2 (image).
	hasCover := len(opts.CoverImageData) > 0
	numRecords := uint16(2) // record 0 + text record
	if hasCover {
		numRecords = 3 // + image record
	}

	// Build EXTH records.
	var exthRecords []exthRecord
	addStringExth := func(recType uint32, val string) {
		if val != "" {
			exthRecords = append(exthRecords, exthRecord{recType: recType, value: []byte(val)})
		}
	}
	addStringExth(exthTitle, title)
	addStringExth(exthAuthor, author)
	addStringExth(exthPublisher, opts.Publisher)
	addStringExth(exthISBN, opts.ISBN)
	addStringExth(exthASIN, opts.ASIN)
	addStringExth(exthLanguage, opts.Language)

	if hasCover {
		// EXTH_COVEROFFSET: offset from FirstImageIndex (0 = first image record).
		exthRecords = append(exthRecords, exthRecord{recType: exthCoverOffset, value: uint32ToBytes(0)})
	}

	// Build EXTH header block.
	exthBlock := buildEXTH(exthRecords)

	// PalmDoc header (16 bytes): no compression, one text record of 4096 bytes.
	textContent := []byte("Hello, world.")
	var pdh [16]byte
	binary.BigEndian.PutUint16(pdh[0:2], 1)                        // Compression: none
	binary.BigEndian.PutUint32(pdh[4:8], uint32(len(textContent))) // TextLength
	binary.BigEndian.PutUint16(pdh[8:10], 1)                       // RecordCount (text records)
	binary.BigEndian.PutUint16(pdh[10:12], 4096)                   // RecordSize
	// Encryption (12:14) and Unknown (14:16) are zero.

	// MOBI header (232 bytes). Field offsets from the mobiHeader struct:
	//   0: Identifier [4]byte    4: HeaderLength     8: MobiType       12: TextEncoding
	//  16: UniqueID             20: FileVersion      24: OrthographicIndex  28: InflectionIndex
	//  32: IndexNames           36: IndexKeys        40-60: ExtraIndex0-5
	//  64: FirstNonBookIndex    68: FullNameOffset    72: FullNameLength  76: Locale
	//  80: InputLanguage        84: OutputLanguage    88: MinVersion      92: FirstImageIndex
	//  96: HuffmanRecordOffset 100: HuffmanRecordCount 104: HuffmanTableOffset 108: HuffmanTableLength
	// 112: ExthFlags           116: Unknown1[32]    148: DrmOffset      152: DrmCount
	// 156: DrmSize             160: DrmFlags        164: Unknown0[12]
	// 176: FirstContentRecordNumber(u16)  178: LastContentRecordNumber(u16)
	// 180: Unknown6            184: FcisRecordIndex 188: FcisRecordCount
	// 192: FlisRecordIndex     196: FlisRecordCount 200-220: various unknowns/srcs
	// 224: ExtraRecordDataFlags 228: IndxRecodOffset
	const mobiHeaderLen = 232
	var mobi [mobiHeaderLen]byte
	copy(mobi[0:4], "MOBI")                                       // Identifier
	binary.BigEndian.PutUint32(mobi[4:8], mobiHeaderLen)          // HeaderLength
	binary.BigEndian.PutUint32(mobi[8:12], 2)                     // MobiType: MOBI book
	binary.BigEndian.PutUint32(mobi[12:16], 65001)                // TextEncoding: UTF-8
	binary.BigEndian.PutUint32(mobi[20:24], 6)                    // FileVersion
	binary.BigEndian.PutUint32(mobi[112:116], 0x40)               // ExthFlags: bit 6 set
	binary.BigEndian.PutUint32(mobi[64:68], uint32(numRecords))   // FirstNonBookIndex
	fullNameOffset := uint32(16 + mobiHeaderLen + len(exthBlock)) // Offset in record 0
	binary.BigEndian.PutUint32(mobi[68:72], fullNameOffset)       // FullNameOffset
	binary.BigEndian.PutUint32(mobi[72:76], uint32(len(title)))   // FullNameLength
	binary.BigEndian.PutUint32(mobi[88:92], 6)                    // MinVersion
	if hasCover {
		binary.BigEndian.PutUint32(mobi[92:96], 2) // FirstImageIndex: record 2
	} else {
		binary.BigEndian.PutUint32(mobi[92:96], math.MaxUint32) // No images
	}

	// Set "unavailable" index fields to 0xFFFFFFFF.
	for _, off := range []int{24, 28, 32, 36, 40, 44, 48, 52, 56, 60} {
		binary.BigEndian.PutUint32(mobi[off:off+4], math.MaxUint32)
	}

	// DRM fields: no DRM.
	binary.BigEndian.PutUint32(mobi[148:152], math.MaxUint32) // DrmOffset
	binary.BigEndian.PutUint32(mobi[152:156], math.MaxUint32) // DrmCount

	// Content record numbers.
	binary.BigEndian.PutUint16(mobi[176:178], 1) // FirstContentRecordNumber
	binary.BigEndian.PutUint16(mobi[178:180], 1) // LastContentRecordNumber

	// FCIS/FLIS: not present.
	binary.BigEndian.PutUint32(mobi[184:188], math.MaxUint32) // FcisRecordIndex
	binary.BigEndian.PutUint32(mobi[188:192], 1)              // FcisRecordCount
	binary.BigEndian.PutUint32(mobi[192:196], math.MaxUint32) // FlisRecordIndex
	binary.BigEndian.PutUint32(mobi[196:200], 1)              // FlisRecordCount

	// IndxRecodOffset: no index.
	binary.BigEndian.PutUint32(mobi[228:232], math.MaxUint32)

	// Assemble record 0: PalmDoc header + MOBI header + EXTH + full name.
	record0 := make([]byte, 0, 16+mobiHeaderLen+len(exthBlock)+len(title))
	record0 = append(record0, pdh[:]...)
	record0 = append(record0, mobi[:]...)
	record0 = append(record0, exthBlock...)
	record0 = append(record0, []byte(title)...)

	// Text record (record 1): minimal content.
	record1 := textContent

	// Assemble records list.
	records := [][]byte{record0, record1}
	if hasCover {
		records = append(records, opts.CoverImageData)
	}

	// Build PalmDB header.
	var pdb [78]byte
	// DatabaseName: up to 31 bytes + null terminator.
	dbName := title
	if len(dbName) > 31 {
		dbName = dbName[:31]
	}
	copy(pdb[0:32], dbName)
	copy(pdb[60:64], "BOOK") // Type
	copy(pdb[64:68], "MOBI") // Creator
	binary.BigEndian.PutUint16(pdb[76:78], numRecords)

	// Record offset table: 8 bytes per record.
	offsetTable := make([]byte, int(numRecords)*8)
	// Data starts after PDB header + offset table + 2-byte padding.
	dataStart := 78 + len(offsetTable) + 2
	currentOffset := dataStart
	for i := range int(numRecords) {
		base := i * 8
		binary.BigEndian.PutUint32(offsetTable[base:base+4], uint32(currentOffset))
		// Attributes (1 byte) + Skip (1 byte) + UniqueID (2 bytes) = 4 bytes, all zero.
		binary.BigEndian.PutUint16(offsetTable[base+6:base+8], uint16(i)) // UniqueID
		currentOffset += len(records[i])
	}

	// Assemble the full file.
	var file []byte
	file = append(file, pdb[:]...)
	file = append(file, offsetTable...)
	file = append(file, 0, 0) // 2-byte padding
	for _, rec := range records {
		file = append(file, rec...)
	}

	return file
}

type exthRecord struct {
	recType uint32
	value   []byte
}

func buildEXTH(records []exthRecord) []byte {
	if len(records) == 0 {
		return nil
	}

	// Calculate total size: 12 (header) + sum of record sizes.
	totalSize := 12
	for _, rec := range records {
		totalSize += 8 + len(rec.value) // type (4) + length (4) + value
	}
	// Pad to 4-byte alignment.
	padding := (4 - totalSize%4) % 4
	totalSize += padding

	buf := make([]byte, totalSize)
	copy(buf[0:4], "EXTH")
	binary.BigEndian.PutUint32(buf[4:8], uint32(totalSize))
	binary.BigEndian.PutUint32(buf[8:12], uint32(len(records)))

	offset := 12
	for _, rec := range records {
		recLen := uint32(8 + len(rec.value))
		binary.BigEndian.PutUint32(buf[offset:offset+4], rec.recType)
		binary.BigEndian.PutUint32(buf[offset+4:offset+8], recLen)
		copy(buf[offset+8:], rec.value)
		offset += int(recLen)
	}

	return buf
}

func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}
