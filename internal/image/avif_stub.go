//go:build noavif

package image

import (
	"errors"
	"image"
)

// encodeAsAVIF is a stub that returns an error when AVIF support is disabled.
// Build with -tags noavif to disable AVIF encoding support.
func encodeAsAVIF(img image.Image, quality int) ([]byte, error) {
	return nil, errors.New("avif encoder disabled (built with -tags noavif)")
}

// isAVIFSupported returns false when AVIF encoding is disabled.
func isAVIFSupported() bool {
	return false
}
