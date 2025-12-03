package image

import (
	"bytes"
	"image"
	"image/png"
)

func EncodeByFormat(img image.Image, format string) ([]byte, string) {
	switch format {
	case "avif":
		if b, err := encodeAsAVIF(img, 75); err == nil && len(b) > 0 {
			return b, "image/avif"
		}
		// Fall through to WebP if AVIF fails
		fallthrough
	case "webp":
		if b, err := encodeAsWebP(img, 85); err == nil && len(b) > 0 {
			return b, "image/webp"
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err == nil {
		return buf.Bytes(), "image/png"
	}
	return nil, ""
}

func ContentTypeFor(format string) string {
	switch format {
	case "avif":
		return "image/avif"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
