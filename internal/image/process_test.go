package image

import (
	"testing"
)

func TestRasterizeSVGWithGradient(t *testing.T) {
	// Test with an SVG that uses linearGradient (like dignitydash favicon)
	gradientSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#6366f1"/>
      <stop offset="100%" style="stop-color:#8b5cf6"/>
    </linearGradient>
  </defs>
  <rect width="64" height="64" rx="14" fill="url(#grad)"/>
</svg>`)

	img, err := RasterizeSVG(gradientSVG, 64, 64)
	if err != nil {
		t.Fatalf("Failed to rasterize gradient SVG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("Expected 64x64, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Check that the image is not blank
	if IsNearlyBlank(img) {
		t.Error("Gradient SVG should not be blank")
	}

	// Check that we have purple/violet colors (from the gradient)
	hasPurple := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 0x8000 {
				continue
			}
			r8, g8, b8 := r>>8, g>>8, b>>8

			// Check for purple/violet tones (blue + red, low green)
			if r8 > 80 && b8 > 180 && g8 < 150 {
				hasPurple = true
				break
			}
		}
		if hasPurple {
			break
		}
	}

	if !hasPurple {
		t.Error("Expected purple/violet gradient colors in the rendered image")
	}
}

func TestRasterizeSVGColorful(t *testing.T) {
	// Test with a colorful SVG
	colorfulSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
  <rect width="100" height="100" fill="#ff0000"/>
  <circle cx="50" cy="50" r="30" fill="#00ff00"/>
  <rect x="35" y="35" width="30" height="30" fill="#0000ff"/>
</svg>`)

	img, err := RasterizeSVG(colorfulSVG, 64, 64)
	if err != nil {
		t.Fatalf("Failed to rasterize colorful SVG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("Expected 64x64, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	if IsNearlyBlank(img) {
		t.Error("Colorful SVG should not be blank")
	}

	// Check for colors
	hasRed := false
	hasGreen := false
	hasBlue := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 0x8000 {
				continue
			}
			r8, g8, b8 := r>>8, g>>8, b>>8

			if r8 > 200 && g8 < 100 && b8 < 100 {
				hasRed = true
			}
			if r8 < 100 && g8 > 200 && b8 < 100 {
				hasGreen = true
			}
			if r8 < 100 && g8 < 100 && b8 > 200 {
				hasBlue = true
			}
		}
	}

	if !hasRed {
		t.Error("Expected red color in the rendered image")
	}
	if !hasGreen {
		t.Error("Expected green color in the rendered image")
	}
	if !hasBlue {
		t.Error("Expected blue color in the rendered image")
	}
}

func TestRasterizeSVGWithCurrentColor(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
  <rect width="100" height="100" fill="white"/>
  <circle cx="50" cy="50" r="40" fill="currentColor"/>
</svg>`)

	img, err := RasterizeSVG(svg, 64, 64)
	if err != nil {
		t.Fatalf("Failed to rasterize SVG with currentColor: %v", err)
	}

	if IsNearlyBlank(img) {
		t.Error("SVG with currentColor should not be blank")
	}
}

func TestIsNearlyBlank(t *testing.T) {
	tests := []struct {
		name     string
		svg      []byte
		expected bool
	}{
		{
			name: "colorful image",
			svg: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
				<rect width="100" height="100" fill="#ff0000"/>
			</svg>`),
			expected: false,
		},
		{
			name: "white image",
			svg: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
				<rect width="100" height="100" fill="white"/>
			</svg>`),
			expected: true,
		},
		{
			name: "black image",
			svg: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
				<rect width="100" height="100" fill="black"/>
			</svg>`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := RasterizeSVG(tt.svg, 64, 64)
			if err != nil {
				t.Fatalf("Failed to rasterize: %v", err)
			}

			result := IsNearlyBlank(img)
			if result != tt.expected {
				t.Errorf("IsNearlyBlank = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFallbackImage(t *testing.T) {
	img, err := CreateFallbackImage(64)
	if err != nil {
		t.Fatalf("Failed to create fallback image: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("Expected 64x64, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	if IsNearlyBlank(img) {
		t.Error("Fallback image should not be blank")
	}
}
