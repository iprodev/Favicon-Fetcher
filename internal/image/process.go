package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
	"golang.org/x/image/draw"
)

// RasterizeSVG converts SVG bytes to a raster image with the specified dimensions.
// Uses tdewolff/canvas for high-quality SVG rendering.
func RasterizeSVG(svgBytes []byte, width, height int) (image.Image, error) {
	// Preprocess SVG to fix common issues
	svgBytes = preprocessSVG(svgBytes)

	// Parse SVG using canvas
	c, err := canvas.ParseSVG(bytes.NewReader(svgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Calculate DPI to get the desired pixel dimensions
	svgW, svgH := c.Size()
	if svgW <= 0 || svgH <= 0 {
		return nil, fmt.Errorf("invalid SVG dimensions: %v x %v", svgW, svgH)
	}

	// Calculate DPI needed to achieve target size
	// canvas uses mm internally, 1 inch = 25.4 mm
	dpiX := float64(width) / (svgW / 25.4)
	dpiY := float64(height) / (svgH / 25.4)
	dpi := dpiX
	if dpiY < dpi {
		dpi = dpiY
	}
	if dpi < 72 {
		dpi = 72
	}
	if dpi > 300 {
		dpi = 300
	}

	// Render to PNG buffer
	var buf bytes.Buffer
	pngRenderer := renderers.PNG(canvas.DPI(dpi))
	if err := c.Write(&buf, pngRenderer); err != nil {
		return nil, fmt.Errorf("failed to render SVG to PNG: %w", err)
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("SVG rendered to empty buffer")
	}

	// Decode PNG
	img, err := png.Decode(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rendered PNG: %w", err)
	}

	// Resize to exact target dimensions if needed
	result := resizeToTarget(img, width, height)

	// Check if the result is usable
	if IsNearlyBlankOrBlack(result) {
		return nil, fmt.Errorf("SVG rendered as blank or black image")
	}

	return result, nil
}

// resizeToTarget resizes an image to fit within the target dimensions with white background.
func resizeToTarget(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	if srcW == width && srcH == height {
		return img
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fillWithWhite(dst)
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// preprocessSVG fixes common SVG issues that cause rendering problems.
func preprocessSVG(data []byte) []byte {
	s := string(data)

	// Ensure SVG has xmlns
	if !strings.Contains(s, "xmlns") && strings.Contains(s, "<svg") {
		s = strings.Replace(s, "<svg", `<svg xmlns="http://www.w3.org/2000/svg"`, 1)
	}

	// Handle currentColor - replace with black as fallback
	s = strings.ReplaceAll(s, "currentColor", "#000000")

	return []byte(s)
}

// fillWithWhite fills an RGBA image with white color.
func fillWithWhite(img *image.RGBA) {
	white := color.RGBA{255, 255, 255, 255}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetRGBA(x, y, white)
		}
	}
}

// IsNearlyBlank checks if an image is mostly transparent.
func IsNearlyBlank(img image.Image) bool {
	if img == nil {
		return true
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stepX := max(w/16, 1)
	stepY := max(h/16, 1)

	nonTransparent := 0
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0x0100 {
				nonTransparent++
				if nonTransparent > 8 {
					return false
				}
			}
		}
	}
	return true
}

// IsNearlyBlankOrBlack checks if an image is mostly transparent, black, or white.
func IsNearlyBlankOrBlack(img image.Image) bool {
	if img == nil {
		return true
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stepX := max(w/16, 1)
	stepY := max(h/16, 1)

	coloredPixels := 0
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r, g, bb, a := img.At(x, y).RGBA()

			// Skip transparent pixels
			if a < 0x8000 {
				continue
			}

			// Normalize to 0-255
			r8, g8, b8 := r>>8, g>>8, bb>>8

			// Check if pixel is not black and not white
			isBlack := r8 < 10 && g8 < 10 && b8 < 10
			isWhite := r8 > 245 && g8 > 245 && b8 > 245

			if !isBlack && !isWhite {
				coloredPixels++
				if coloredPixels > 5 {
					return false
				}
			}
		}
	}
	return coloredPixels <= 5
}

// ResizeImage resizes an image to the target size using high-quality interpolation.
func ResizeImage(img image.Image, size int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

// ResizeImageWithBackground resizes an image with a background color.
func ResizeImageWithBackground(img image.Image, size int, bgColor color.Color) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.Set(x, y, bgColor)
		}
	}
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

// FallbackGlobeSVG returns a simple globe SVG for fallback.
func FallbackGlobeSVG(size int) []byte {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 100 100">
  <rect width="100" height="100" fill="white"/>
  <circle cx="50" cy="50" r="45" fill="#e3f2fd" stroke="#1976d2" stroke-width="2"/>
  <ellipse cx="50" cy="50" rx="45" ry="20" fill="none" stroke="#1976d2" stroke-width="1"/>
  <ellipse cx="50" cy="50" rx="20" ry="45" fill="none" stroke="#1976d2" stroke-width="1"/>
  <line x1="5" y1="50" x2="95" y2="50" stroke="#1976d2" stroke-width="1"/>
  <line x1="50" y1="5" x2="50" y2="95" stroke="#1976d2" stroke-width="1"/>
  <path d="M15 35 Q50 25 85 35" fill="none" stroke="#4caf50" stroke-width="2"/>
  <path d="M10 65 Q50 75 90 65" fill="none" stroke="#4caf50" stroke-width="2"/>
</svg>`, size, size)
	return []byte(svg)
}

// CreateFallbackImage creates a fallback globe image.
func CreateFallbackImage(size int) (image.Image, error) {
	svgBytes := FallbackGlobeSVG(size)
	img, err := RasterizeSVG(svgBytes, size, size)
	if err != nil {
		// Ultimate fallback: create a simple colored image
		return createSimpleFallback(size), nil
	}
	return img, nil
}

// createSimpleFallback creates a simple fallback image without SVG.
func createSimpleFallback(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Light blue background
	bgColor := color.RGBA{227, 242, 253, 255}
	borderColor := color.RGBA{25, 118, 210, 255}

	// Fill background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, bgColor)
		}
	}

	// Draw simple circle border
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size) * 0.45

	for angle := 0.0; angle < 360; angle += 0.5 {
		rad := angle * 3.14159265 / 180
		x := int(cx + r*cos(rad))
		y := int(cy + r*sin(rad))
		if x >= 0 && x < size && y >= 0 && y < size {
			img.SetRGBA(x, y, borderColor)
		}
	}

	return img
}

func cos(rad float64) float64 {
	// Taylor series approximation
	rad = mod2pi(rad)
	return 1 - rad*rad/2 + rad*rad*rad*rad/24 - rad*rad*rad*rad*rad*rad/720
}

func sin(rad float64) float64 {
	rad = mod2pi(rad)
	return rad - rad*rad*rad/6 + rad*rad*rad*rad*rad/120 - rad*rad*rad*rad*rad*rad*rad/5040
}

func mod2pi(x float64) float64 {
	const twoPi = 6.28318530718
	for x > twoPi {
		x -= twoPi
	}
	for x < 0 {
		x += twoPi
	}
	return x
}

// CreateBlankImage creates a 1x1 white image.
func CreateBlankImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{255, 255, 255, 255})
	return img
}

// EnsureOpaque converts an image to have an opaque white background.
func EnsureOpaque(img image.Image) image.Image {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	white := color.RGBA{255, 255, 255, 255}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.SetRGBA(x, y, white)
		}
	}

	draw.Draw(rgba, bounds, img, bounds.Min, draw.Over)
	return rgba
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
