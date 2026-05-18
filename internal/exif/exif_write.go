package exif

import (
	"bytes"
	"errors"
)

func splitReadyToken(data []byte, atEOF bool) (int, []byte, error) {
	idx := bytes.Index(data, readyToken)
	if idx == -1 {
		if atEOF && len(data) > 0 {
			return 0, data, errors.New("no final token found")
		}

		return 0, nil, nil
	}

	return idx + readyTokenLen, data[:idx], nil
}
