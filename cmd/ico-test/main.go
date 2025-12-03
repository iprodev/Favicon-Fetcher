package main

import (
	"flag"
	"fmt"
	"os"

	"faviconsvc/internal/image"
)

// Simple CLI tool to test ICO decoding
// Usage: go run cmd/ico-test/main.go -file path/to/favicon.ico

func main() {
	filePath := flag.String("file", "", "Path to ICO file to test")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: go run cmd/ico-test/main.go -file path/to/favicon.ico")
		os.Exit(1)
	}

	// Read ICO file
	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing ICO file: %s (%d bytes)\n", *filePath, len(data))

	// Try to decode
	img, err := image.DecodeICOSelectLargest(data)
	if err != nil {
		fmt.Printf("❌ Decode failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully decoded!\n")
	bounds := img.Bounds()
	fmt.Printf("   Size: %dx%d\n", bounds.Dx(), bounds.Dy())

	// Check if blank
	if image.IsNearlyBlank(img) {
		fmt.Printf("⚠️  Warning: Image appears nearly blank (might be transparency issue)\n")
	}

	// Test resize
	fmt.Printf("\n🔄 Testing resize to 32x32...\n")
	resized := image.ResizeImage(img, 32)
	fmt.Printf("✅ Resize successful: %dx%d\n", resized.Bounds().Dx(), resized.Bounds().Dy())

	// Try encoding to PNG
	fmt.Printf("\n💾 Testing PNG encoding...\n")
	pngData, contentType := image.EncodeByFormat(resized, "png")
	if len(pngData) > 0 {
		fmt.Printf("✅ PNG encoding successful: %d bytes, type=%s\n", len(pngData), contentType)
		
		// Optionally save output
		outPath := *filePath + ".test.png"
		if err := os.WriteFile(outPath, pngData, 0644); err == nil {
			fmt.Printf("💾 Saved test output to: %s\n", outPath)
		}
	} else {
		fmt.Printf("❌ PNG encoding failed\n")
	}

	fmt.Printf("\n✨ All tests passed!\n")
}
