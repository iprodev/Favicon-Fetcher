//go:build nowebp

package image

import (
	"errors"
	"image"
)

func encodeAsWebP(img image.Image, quality int) ([]byte, error) {
	return nil, errors.New("webp encoder disabled (built with -tags nowebp)")
}
