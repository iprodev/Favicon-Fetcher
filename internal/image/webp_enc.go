//go:build !nowebp

package image

import (
	"bytes"
	"image"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

func encodeAsWebP(img image.Image, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = 85
	}
	q := float32(quality)
	opts, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, q)
	if err != nil {
		return nil, err
	}
	opts.Method = 4
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
