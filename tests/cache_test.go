package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"faviconsvc/internal/cache"
)

func TestCacheBasicOperations(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)

	if err := cm.EnsureDirs(); err != nil {
		t.Fatalf("Failed to create cache dirs: %v", err)
	}

	// Test write and read
	testURL := "https://example.com/favicon.ico"
	testData := []byte("test favicon data")

	if err := cm.WriteOrigToCache(testURL, testData); err != nil {
		t.Fatalf("Failed to write to cache: %v", err)
	}

	readData, ok := cm.ReadOrigFromCache(testURL)
	if !ok {
		t.Fatal("Failed to read from cache")
	}

	if string(readData) != string(testData) {
		t.Errorf("Read data mismatch: got %s, want %s", readData, testData)
	}
}

func TestCacheMeta(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)

	if err := cm.EnsureDirs(); err != nil {
		t.Fatalf("Failed to create cache dirs: %v", err)
	}

	testURL := "https://example.com/favicon.ico"
	meta := cache.OrigMeta{
		URL:          testURL,
		ETag:         "test-etag",
		LastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
		UpdatedAt:    time.Now(),
	}

	if err := cm.WriteOrigMeta(testURL, meta); err != nil {
		t.Fatalf("Failed to write meta: %v", err)
	}

	readMeta, ok := cm.ReadOrigMeta(testURL)
	if !ok {
		t.Fatal("Failed to read meta")
	}

	if readMeta.ETag != meta.ETag {
		t.Errorf("ETag mismatch: got %s, want %s", readMeta.ETag, meta.ETag)
	}
}

func TestCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Millisecond)

	if err := cm.EnsureDirs(); err != nil {
		t.Fatalf("Failed to create cache dirs: %v", err)
	}

	testURL := "https://example.com/favicon.ico"
	testData := []byte("test data")

	if err := cm.WriteOrigToCache(testURL, testData); err != nil {
		t.Fatalf("Failed to write to cache: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	_, ok := cm.ReadOrigFromCache(testURL)
	if ok {
		t.Error("Cache should have expired")
	}
}

func TestResizedCache(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)

	if err := cm.EnsureDirs(); err != nil {
		t.Fatalf("Failed to create cache dirs: %v", err)
	}

	testURL := "https://example.com/favicon.ico"
	testData := []byte("resized data")
	size := 32
	format := "png"

	if err := cm.WriteResizedToCache(testURL, size, format, testData); err != nil {
		t.Fatalf("Failed to write resized cache: %v", err)
	}

	readData, ok, _ := cm.ReadResizedFromCacheWithMod(testURL, size, format)
	if !ok {
		t.Fatal("Failed to read resized cache")
	}

	if string(readData) != string(testData) {
		t.Errorf("Data mismatch: got %s, want %s", readData, testData)
	}
}

func TestCachePaths(t *testing.T) {
	tmpDir := t.TempDir()
	cm := cache.New(tmpDir, 1*time.Hour)

	origDir := cm.OrigCacheDir()
	if !filepath.IsAbs(origDir) {
		t.Error("OrigCacheDir should return absolute path")
	}
	if filepath.Base(origDir) != "orig" {
		t.Errorf("Expected orig directory, got %s", filepath.Base(origDir))
	}

	resizedDir := cm.ResizedCacheDir()
	if filepath.Base(resizedDir) != "resized" {
		t.Errorf("Expected resized directory, got %s", filepath.Base(resizedDir))
	}
}
