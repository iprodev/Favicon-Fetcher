package image

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sort"

	"github.com/gen2brain/avif"
	ico "github.com/sergeymakinen/go-ico"
	"golang.org/x/image/bmp"
	xwebp "golang.org/x/image/webp"
)

func DecodeICOSelectLargest(b []byte) (image.Image, error) {
	if len(b) < 6 {
		return nil, errors.New("ico: too small")
	}

	r := bytes.NewReader(b)
	var reserved, icotype, count uint16
	_ = binary.Read(r, binary.LittleEndian, &reserved)
	_ = binary.Read(r, binary.LittleEndian, &icotype)
	_ = binary.Read(r, binary.LittleEndian, &count)

	if icotype != 1 || count == 0 {
		img, err := ico.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	type entry struct {
		w, h         int
		size, offset uint32
		isPNG        bool
		bpp          int // bits per pixel
	}
	entries := make([]entry, 0, count)

	for i := 0; i < int(count); i++ {
		var e [16]byte
		if _, err := io.ReadFull(r, e[:]); err != nil {
			break
		}
		w := int(e[0])
		h := int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		bpp := int(e[6]) // bits per pixel
		if bpp == 0 {
			bpp = 32 // assume 32-bit if not specified
		}
		size := binary.LittleEndian.Uint32(e[8:12])
		offset := binary.LittleEndian.Uint32(e[12:16])
		entries = append(entries, entry{w: w, h: h, size: size, offset: offset, bpp: bpp})
	}

	if len(entries) == 0 {
		return ico.Decode(bytes.NewReader(b))
	}

	// Check which entries are PNG
	for i := range entries {
		e := &entries[i]
		if int(e.offset+e.size) > len(b) || e.size == 0 {
			continue
		}
		slice := b[e.offset : e.offset+e.size]
		if len(slice) >= 8 && bytes.Equal(slice[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
			e.isPNG = true
		}
	}

	// Sort by quality: PNG > size > bit depth
	sort.Slice(entries, func(i, j int) bool {
		// Prioritize PNG over BMP
		if entries[i].isPNG != entries[j].isPNG {
			return entries[i].isPNG
		}
		// Then by size
		sizeI := entries[i].w * entries[i].h
		sizeJ := entries[j].w * entries[j].h
		if sizeI != sizeJ {
			return sizeI > sizeJ
		}
		// Finally by bit depth (higher is better)
		return entries[i].bpp > entries[j].bpp
	})

	// Try to decode in priority order
	for _, e := range entries {
		if int(e.offset+e.size) > len(b) || e.size == 0 {
			continue
		}
		slice := b[e.offset : e.offset+e.size]

		// Try PNG first
		if e.isPNG {
			if img, err := png.Decode(bytes.NewReader(slice)); err == nil {
				return img, nil
			}
		}
		
		// Try BMP (might not have alpha channel)
		if img, err := bmp.Decode(bytes.NewReader(slice)); err == nil {
			// BMP in ICO doesn't handle transparency well
			// Check if it looks blank and skip if so
			if !IsNearlyBlank(img) {
				return img, nil
			}
		}
	}

	return ico.Decode(bytes.NewReader(b))
}

func DecodeImageRasterOnly(b []byte) (image.Image, error) {
	if img, err := png.Decode(bytes.NewReader(b)); err == nil {
		return img, nil
	}
	if img, err := jpeg.Decode(bytes.NewReader(b)); err == nil {
		return img, nil
	}
	if img, err := gif.Decode(bytes.NewReader(b)); err == nil {
		return img, nil
	}
	if img, err := xwebp.Decode(bytes.NewReader(b)); err == nil {
		return img, nil
	}
	if img, err := avif.Decode(bytes.NewReader(b)); err == nil {
		return img, nil
	}
	return nil, errors.New("unsupported raster format")
}
