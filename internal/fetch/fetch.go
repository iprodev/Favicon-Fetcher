package fetch

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"faviconsvc/internal/security"
	"faviconsvc/pkg/logger"
)

const (
	MaxFetchBytes = 4 << 20 // 4MB
	MaxHTMLBytes  = 1 << 20 // 1MB
	UABrowser     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
)

var HTTPClient *http.Client

func InitHTTPClient() {
	HTTPClient = &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			DialContext:         security.ValidatedDialContext,
			ForceAttemptHTTP2:   true,
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConnsPerHost: 4,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 8 {
				return errors.New("too many redirects")
			}
			if !security.IsAllowedScheme(req.URL) {
				return errors.New("blocked redirect scheme")
			}
			return nil
		},
	}
}

func FetchURLFull(ctx context.Context, canonURL string) ([]byte, string, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonURL, nil)
	if err != nil {
		return nil, "", "", "", err
	}
	req.Header.Set("User-Agent", UABrowser)
	req.Header.Set("Accept", "image/*,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip")

	logger.Debug("Fetching URL: %s", canonURL)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		logger.Warn("Fetch failed for %s: %v", canonURL, err)
		return nil, "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("Fetch got status %d for %s", resp.StatusCode, canonURL)
		return nil, "", "", "", errors.New("status " + resp.Status)
	}

	body, err := readPossiblyGzipped(resp)
	if err != nil {
		return nil, "", "", "", err
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(peek512(body))
	}
	etag := strings.TrimSpace(resp.Header.Get("ETag"))
	lastMod := strings.TrimSpace(resp.Header.Get("Last-Modified"))

	logger.Debug("Fetched %s: %d bytes, content-type: %s", canonURL, len(body), ct)
	return body, ct, etag, lastMod, nil
}

func FetchURLConditional(ctx context.Context, canonURL string, etag, lastMod string) ([]byte, string, int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonURL, nil)
	if err != nil {
		return nil, "", 0, "", "", err
	}
	req.Header.Set("User-Agent", UABrowser)
	req.Header.Set("Accept", "image/*,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip")

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastMod != "" {
		req.Header.Set("If-Modified-Since", lastMod)
	}

	logger.Debug("Conditional fetch for %s (ETag: %s, LastMod: %s)", canonURL, etag, lastMod)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, "", 0, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		logger.Debug("Cache hit (304) for %s", canonURL)
		return nil, "", 304, etag, lastMod, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", resp.StatusCode, "", "", errors.New("status " + resp.Status)
	}

	body, err := readPossiblyGzipped(resp)
	if err != nil {
		return nil, "", resp.StatusCode, "", "", err
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(peek512(body))
	}
	newETag := strings.TrimSpace(resp.Header.Get("ETag"))
	newLM := strings.TrimSpace(resp.Header.Get("Last-Modified"))

	logger.Debug("Fetched (conditional) %s: %d bytes", canonURL, len(body))
	return body, ct, resp.StatusCode, newETag, newLM, nil
}

func readPossiblyGzipped(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		reader = zr
	}
	lr := io.LimitReader(reader, MaxFetchBytes)
	return io.ReadAll(lr)
}

func peek512(b []byte) []byte {
	if len(b) > 512 {
		return b[:512]
	}
	return b
}
