package image

import (
	"bytes"
	"image"

	"github.com/HugoSmits86/nativewebp"
)

func encodeAsWebP(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer

	// nativewebp.Encode uses lossless encoding by default
	// Pass nil for default options
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
