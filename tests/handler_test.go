package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"faviconsvc/internal/cache"
	"faviconsvc/internal/fetch"
	"faviconsvc/internal/handler"
)

func TestFaviconHandler_NoURL(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		1*time.Hour,
		1*time.Hour,
		true,
	)

	req := httptest.NewRequest("GET", "/favicons", nil)
	w := httptest.NewRecorder()

	handler.FaviconHandler(cfg)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "image/png" && contentType != "image/webp" {
		t.Errorf("Expected image content type, got %s", contentType)
	}
}

func TestFaviconHandler_WithSize(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		1*time.Hour,
		1*time.Hour,
		true,
	)

	tests := []struct {
		size     string
		wantCode int
	}{
		{"32", 200},
		{"64", 200},
		{"16", 200},
		{"256", 200},
		{"512", 200}, // Should be capped to 256
		{"8", 200},   // Should be raised to 16
	}

	for _, tt := range tests {
		t.Run("size="+tt.size, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/favicons?sz="+tt.size, nil)
			w := httptest.NewRecorder()

			handler.FaviconHandler(cfg)(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func TestFaviconHandler_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		1*time.Hour,
		1*time.Hour,
		true,
	)

	tests := []string{
		"localhost",
		"127.0.0.1",
		"http://10.0.0.1",
		"ftp://example.com",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/favicons?url="+url, nil)
			w := httptest.NewRecorder()

			handler.FaviconHandler(cfg)(w, req)

			// Should return fallback image
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestFaviconHandler_ETag(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		1*time.Hour,
		1*time.Hour,
		true,
	)

	// First request
	req1 := httptest.NewRequest("GET", "/favicons", nil)
	w1 := httptest.NewRecorder()
	handler.FaviconHandler(cfg)(w1, req1)

	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("Expected ETag header")
	}

	// Second request with If-None-Match
	req2 := httptest.NewRequest("GET", "/favicons", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	handler.FaviconHandler(cfg)(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("Expected status 304, got %d", w2.Code)
	}
}

func TestFaviconHandler_CacheHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		2*time.Hour,
		3*time.Hour,
		true,
	)

	req := httptest.NewRequest("GET", "/favicons", nil)
	w := httptest.NewRecorder()

	handler.FaviconHandler(cfg)(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Error("Expected Cache-Control header")
	}

	if w.Header().Get("Expires") == "" {
		t.Error("Expected Expires header")
	}
}

func TestFaviconHandler_WebPAccept(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)
	_ = cm.EnsureDirs()

	fetch.InitHTTPClient()

	cfg := handler.NewConfig(
		cm,
		1*time.Hour,
		1*time.Hour,
		true,
	)

	req := httptest.NewRequest("GET", "/favicons", nil)
	req.Header.Set("Accept", "image/webp,image/png")
	w := httptest.NewRecorder()

	handler.FaviconHandler(cfg)(w, req)

	// Note: WebP might not be available depending on build tags
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/webp" && contentType != "image/png" {
		t.Errorf("Unexpected content type: %s", contentType)
	}
}
