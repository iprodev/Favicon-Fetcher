//go:build !noavif

package image

import (
	"bytes"
	"image"

	"github.com/gen2brain/avif"
)

// encodeAsAVIF encodes an image to AVIF format.
func encodeAsAVIF(img image.Image, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = 75
	}
	if quality > 100 {
		quality = 100
	}

	var buf bytes.Buffer

	opts := avif.Options{
		Quality: quality,
		Speed:   6, // 0-10, higher is faster
	}

	if err := avif.Encode(&buf, img, opts); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// isAVIFSupported returns true when AVIF encoding is available.
func isAVIFSupported() bool {
	return true
}
