package image

import (
	"image"
	"image/color"
	"testing"
)

func TestEncodeByFormat_AVIF(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Test AVIF encoding
	data, contentType := EncodeByFormat(img, "avif")

	if isAVIFSupported() {
		// AVIF is available
		if data == nil {
			t.Error("Expected AVIF data, got nil")
		}
		if contentType != "image/avif" {
			t.Errorf("Expected content type image/avif, got %s", contentType)
		}
		// AVIF files start with specific bytes (ftyp box)
		if len(data) > 8 && string(data[4:8]) != "ftyp" {
			t.Error("AVIF data doesn't appear to be valid")
		}
	} else {
		// AVIF not available, should fallback to WebP or PNG
		if data == nil {
			t.Error("Expected fallback encoding, got nil")
		}
		if contentType != "image/webp" && contentType != "image/png" {
			t.Errorf("Expected fallback to webp or png, got %s", contentType)
		}
	}
}

func TestContentTypeFor_AVIF(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"avif", "image/avif"},
		{"webp", "image/webp"},
		{"png", "image/png"},
		{"", "image/png"},
		{"unknown", "image/png"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := ContentTypeFor(tt.format)
			if got != tt.expected {
				t.Errorf("ContentTypeFor(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestIsAVIFSupported(t *testing.T) {
	// Just ensure the function doesn't panic
	supported := isAVIFSupported()
	t.Logf("AVIF support: %v", supported)
}

func BenchmarkEncodeAVIF(b *testing.B) {
	if !isAVIFSupported() {
		b.Skip("AVIF not supported")
	}

	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 4),
				G: uint8(y * 4),
				B: 128,
				A: 255,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodeAsAVIF(img, 75)
	}
}
