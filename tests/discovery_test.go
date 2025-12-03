package tests

import (
	"testing"

	"faviconsvc/internal/discovery"
)

func TestIsICO(t *testing.T) {
	tests := []struct {
		contentType string
		url         string
		want        bool
	}{
		{"image/x-icon", "test.png", true},
		{"image/vnd.microsoft.icon", "test.png", true},
		{"image/png", "test.ico", true},
		{"image/png", "test.png", false},
		{"", "favicon.ico", true},
		{"", "image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := discovery.IsICO(tt.contentType, tt.url)
			if got != tt.want {
				t.Errorf("IsICO(%q, %q) = %v, want %v", tt.contentType, tt.url, got, tt.want)
			}
		})
	}
}

func TestIsSVGContentType(t *testing.T) {
	tests := []struct {
		contentType string
		url         string
		want        bool
	}{
		{"image/svg+xml", "test.png", true},
		{"image/png", "test.svg", true},
		{"image/png", "test.png", false},
		{"", "icon.svg", true},
		{"", "icon.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := discovery.IsSVGContentType(tt.contentType, tt.url)
			if got != tt.want {
				t.Errorf("IsSVGContentType(%q, %q) = %v, want %v", tt.contentType, tt.url, got, tt.want)
			}
		})
	}
}

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		want        bool
	}{
		{"DOCTYPE HTML", []byte("<!doctype html><html></html>"), "", true},
		{"HTML tag", []byte("<html><head></head></html>"), "", true},
		{"With whitespace", []byte("  \n  <!DOCTYPE HTML>"), "", true},
		{"JSON data", []byte(`{"test": "data"}`), "", false},
		{"Binary data", []byte{0x89, 0x50, 0x4e, 0x47}, "", false},
		{"Content-Type HTML", []byte("test"), "text/html", true},
		{"Content-Type JSON", []byte("test"), "application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discovery.LooksLikeHTML(tt.data, tt.contentType)
			if got != tt.want {
				t.Errorf("LooksLikeHTML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanonicalizeURLString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"https://Example.COM/Path",
			"https://example.com/Path",
		},
		{
			"https://example.com:443/path",
			"https://example.com/path",
		},
		{
			"http://example.com:80/path",
			"http://example.com/path",
		},
		{
			"https://example.com/path?b=2&a=1",
			"https://example.com/path?a=1&b=2",
		},
		{
			"https://example.com#fragment",
			"https://example.com/",
		},
		{
			"https://example.com",
			"https://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := discovery.CanonicalizeURLString(tt.input)
			if got != tt.want {
				t.Errorf("CanonicalizeURLString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSizes(t *testing.T) {
	// This would need to be exported from discovery package or tested indirectly
	// For now, we test the overall behavior through integration tests
}

func TestComputeSizeScore(t *testing.T) {
	// Similar to above - internal function
}
