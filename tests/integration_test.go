// +build integration

package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	testServerAddr = "http://localhost:19090"
	serverStartTimeout = 10 * time.Second
)

var (
	serverCmd *exec.Cmd
	serverReady bool
)

// TestMain sets up and tears down the test server
func TestMain(m *testing.M) {
	// Build the server
	if err := buildServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build server: %v\n", err)
		os.Exit(1)
	}

	// Start the server
	if err := startServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Wait for server to be ready
	if !waitForServer() {
		fmt.Fprintf(os.Stderr, "Server failed to start within timeout\n")
		stopServer()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	stopServer()
	cleanupCache()

	os.Exit(code)
}

func buildServer() error {
	fmt.Println("Building server...")
	cmd := exec.Command("go", "build", "-o", "test-server", "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startServer() error {
	fmt.Println("Starting test server...")
	cacheDir := filepath.Join(os.TempDir(), "favicon-test-cache")
	os.MkdirAll(cacheDir, 0755)

	serverCmd = exec.Command("./test-server",
		"-port", "19090",
		"-cache-dir", cacheDir,
		"-cache-ttl", "1h",
		"-log-level", "debug",
	)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func stopServer() {
	if serverCmd != nil && serverCmd.Process != nil {
		fmt.Println("Stopping test server...")
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}
	os.Remove("./test-server")
}

func cleanupCache() {
	cacheDir := filepath.Join(os.TempDir(), "favicon-test-cache")
	os.RemoveAll(cacheDir)
}

func waitForServer() bool {
	fmt.Println("Waiting for server to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), serverStartTimeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			resp, err := http.Get(testServerAddr + "/health")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				fmt.Println("Server is ready!")
				serverReady = true
				return true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// Integration Tests

func TestIntegration_HealthCheck(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	resp, err := http.Get(testServerAddr + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("Unexpected body: %s", body)
	}
}

func TestIntegration_FetchFavicon_NoDomain(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	resp, err := http.Get(testServerAddr + "/favicons")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return fallback image
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/png" && contentType != "image/webp" {
		t.Errorf("Expected image content type, got %s", contentType)
	}
}

func TestIntegration_FetchFavicon_WithDomain(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	// Using example.com as it's stable and has a favicon
	resp, err := http.Get(testServerAddr + "/favicons?domain=example.com")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/png" && contentType != "image/webp" {
		t.Errorf("Expected image content type, got %s", contentType)
	}

	// Check cache headers
	if resp.Header.Get("Cache-Control") == "" {
		t.Error("Missing Cache-Control header")
	}
	if resp.Header.Get("ETag") == "" {
		t.Error("Missing ETag header")
	}
}

func TestIntegration_FetchFavicon_WithSize(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	sizes := []string{"16", "32", "64", "128"}
	for _, size := range sizes {
		t.Run("size="+size, func(t *testing.T) {
			url := fmt.Sprintf("%s/favicons?domain=example.com&sz=%s", testServerAddr, size)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestIntegration_FetchFavicon_InvalidURL(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	invalidURLs := []string{
		"localhost",
		"127.0.0.1",
		"10.0.0.1",
		"file:///etc/passwd",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/favicons?url=%s", testServerAddr, url))
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Should return fallback image, not error
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 (fallback), got %d", resp.StatusCode)
			}
		})
	}
}

func TestIntegration_ETag_NotModified(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	// First request
	resp1, err := http.Get(testServerAddr + "/favicons?domain=example.com")
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	etag := resp1.Header.Get("ETag")
	if etag == "" {
		t.Fatal("No ETag in first response")
	}

	// Second request with If-None-Match
	req, _ := http.NewRequest("GET", testServerAddr+"/favicons?domain=example.com", nil)
	req.Header.Set("If-None-Match", etag)

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotModified {
		t.Errorf("Expected status 304, got %d", resp2.StatusCode)
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	// Test thundering herd protection
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			resp, err := http.Get(testServerAddr + "/favicons?domain=example.com")
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			} else {
				resp.Body.Close()
			}
			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestIntegration_WebPAccept(t *testing.T) {
	if !serverReady {
		t.Skip("Server not ready")
	}

	req, _ := http.NewRequest("GET", testServerAddr+"/favicons?domain=example.com", nil)
	req.Header.Set("Accept", "image/webp,image/png")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	// Might be webp or png depending on build
	if contentType != "image/webp" && contentType != "image/png" {
		t.Errorf("Unexpected content type: %s", contentType)
	}
}

func TestIntegration_LargeNumberOfRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	if !serverReady {
		t.Skip("Server not ready")
	}

	// Stress test
	numRequests := 100
	done := make(chan bool, numRequests)
	errors := 0

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(i int) {
			url := fmt.Sprintf("%s/favicons?domain=example.com&sz=%d", testServerAddr, 16+(i%4)*16)
			resp, err := http.Get(url)
			if err != nil || resp.StatusCode != 200 {
				errors++
			}
			if resp != nil {
				resp.Body.Close()
			}
			done <- true
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("Completed %d requests in %v (%.2f req/s)", numRequests, elapsed, float64(numRequests)/elapsed.Seconds())

	if errors > 0 {
		t.Errorf("Failed %d out of %d requests", errors, numRequests)
	}
}
