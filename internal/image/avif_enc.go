//go:build !noavif

package image

import (
	"bytes"
	"image"

	"github.com/gen2brain/avif"
)

// encodeAsAVIF encodes an image to AVIF format with the specified quality.
// Quality ranges from 0 (worst) to 100 (best). Default is 75.
// AVIF typically provides 20-30% better compression than WebP.
func encodeAsAVIF(img image.Image, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = 75
	}
	if quality > 100 {
		quality = 100
	}

	opts := avif.Options{
		Quality:           quality,
		QualityAlpha:      quality,
		Speed:             6, // 0-10, higher is faster but lower quality
		ChromaSubsampling: avif.YUV420, // Best compression for icons
	}

	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isAVIFSupported returns true when AVIF encoding is available.
func isAVIFSupported() bool {
	return true
}
