// Package handler provides HTTP request handlers for the favicon service.
// It handles favicon fetching, caching, and serving with support for
// multiple formats, sizes, and HTTP caching mechanisms.
package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"

	"faviconsvc/internal/cache"
	"faviconsvc/internal/discovery"
	"faviconsvc/internal/fetch"
	imgpkg "faviconsvc/internal/image"
	"faviconsvc/internal/security"
	"faviconsvc/pkg/logger"
)

const (
	DefaultSize = 32
	MinSize     = 16
	MaxSize     = 256
)

// Config holds configuration for the favicon handler.
// It includes cache management, HTTP caching headers, and request deduplication.
type Config struct {
	CacheManager    *cache.Manager
	BrowserMaxAge   time.Duration
	CDNSMaxAge      time.Duration
	UseETag         bool
	fetchGroup      *cache.Group // Prevents thundering herd
}

// NewConfig creates a new handler configuration with the specified settings.
// It also initializes the singleflight group for request deduplication.
func NewConfig(cm *cache.Manager, browserMaxAge, cdnSMaxAge time.Duration, useETag bool) *Config {
	return &Config{
		CacheManager:  cm,
		BrowserMaxAge: browserMaxAge,
		CDNSMaxAge:    cdnSMaxAge,
		UseETag:       useETag,
		fetchGroup:    cache.NewGroup(),
	}
}

// FaviconHandler returns an HTTP handler function that processes favicon requests.
// It handles URL parsing, size validation, format negotiation, icon discovery,
// and response generation with appropriate caching headers.
//
// Query parameters:
//   - url or domain: Website URL or domain name (required)
//   - sz or size: Output size in pixels (16-256, default: 32)
//
// Response headers:
//   - Content-Type: image/png or image/webp
//   - Cache-Control: Public caching directives
//   - ETag: Entity tag for conditional requests
//   - Last-Modified: Last modification time
//   - Expires: Cache expiration time
func FaviconHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse size parameter
		szStr := r.URL.Query().Get("sz")
		if szStr == "" {
			szStr = r.URL.Query().Get("size")
		}
		size := DefaultSize
		if n, err := strconv.Atoi(szStr); err == nil {
			if n < MinSize {
				n = MinSize
			}
			if n > MaxSize {
				n = MaxSize
			}
			size = n
		}

		// Determine output format
		wantFormat := pickFormatByAccept(r.Header.Get("Accept"))

		// Parse URL parameter
		pageURL := strings.TrimSpace(r.URL.Query().Get("url"))
		if pageURL == "" {
			if d := strings.TrimSpace(r.URL.Query().Get("domain")); d != "" {
				pageURL = "https://" + d
			}
		}

		if pageURL == "" {
			serveImageVariant(w, r, nil, size, wantFormat, time.Now(), cfg)
			return
		}

		u, err := security.NormalizeURL(pageURL)
		if err != nil {
			logger.Warn("Invalid URL '%s': %v", pageURL, err)
			serveImageVariant(w, r, nil, size, wantFormat, time.Now(), cfg)
			return
		}

		// Discover and fetch icons
		candidates := discovery.DiscoverFromPageThenRoot(ctx, u, size)
		var best image.Image
		var bestArea int64 = -1
		var bestSrc string

		for _, cand := range candidates {
			iconURL := cand.URL
			origBytes, ct, err := fetchURLCachedWithRevalidation(ctx, iconURL, cfg)
			if err != nil || len(origBytes) == 0 || discovery.LooksLikeHTML(origBytes, ct) {
				continue
			}

			var img image.Image
			var area int64

			if discovery.IsSVGContentType(ct, iconURL) {
				img, err = imgpkg.RasterizeSVG(origBytes, size, size)
				if err != nil {
					logger.Debug("SVG rasterization failed for %s: %v", iconURL, err)
					continue
				}
				if imgpkg.IsNearlyBlankOrBlack(img) {
					logger.Debug("SVG rendered as blank/black for %s, skipping", iconURL)
					continue
				}
				area = 1 << 50 // SVG priority
			} else if discovery.IsICO(ct, iconURL) {
				img, err = imgpkg.DecodeICOSelectLargest(origBytes)
				if err != nil {
					continue
				}
				area = int64(img.Bounds().Dx()) * int64(img.Bounds().Dy())
			} else {
				img, err = imgpkg.DecodeImageRasterOnly(origBytes)
				if err != nil {
					continue
				}
				area = int64(img.Bounds().Dx()) * int64(img.Bounds().Dy())
			}

			dst := imgpkg.ResizeImage(img, size)
			if area > bestArea {
				bestArea, best, bestSrc = area, dst, iconURL
			}
		}

		if best == nil {
			serveImageVariant(w, r, nil, size, wantFormat, time.Now(), cfg)
			return
		}

		serveImageVariantWithSource(w, r, best, size, wantFormat, time.Now(), bestSrc, cfg)
	}
}

func serveImageVariantWithSource(w http.ResponseWriter, r *http.Request, img image.Image, size int, format string, lastMod time.Time, srcURL string, cfg *Config) {
	// Try cache first
	if b, ok, mod := cfg.CacheManager.ReadResizedFromCacheWithMod(srcURL, size, format); ok && len(b) > 0 {
		serveBytes(w, r, b, imgpkg.ContentTypeFor(format), mod, cfg)
		return
	}

	// Encode
	data, ct := imgpkg.EncodeByFormat(img, format)
	if data == nil {
		data, ct = imgpkg.EncodeByFormat(img, "png")
	}
	if len(data) == 0 {
		var buf bytes.Buffer
		_ = png.Encode(&buf, imgpkg.CreateBlankImage())
		data, ct = buf.Bytes(), "image/png"
	}

	_ = cfg.CacheManager.WriteResizedToCache(srcURL, size, format, data)
	serveBytes(w, r, data, ct, lastMod, cfg)
}

func serveImageVariant(w http.ResponseWriter, r *http.Request, img image.Image, size int, format string, lastMod time.Time, cfg *Config) {
	if img == nil {
		var err error
		img, err = imgpkg.CreateFallbackImage(size)
		if err != nil {
			img = imgpkg.CreateBlankImage()
		}
	}

	data, ct := imgpkg.EncodeByFormat(img, format)
	if data == nil {
		data, ct = imgpkg.EncodeByFormat(img, "png")
	}
	if len(data) == 0 {
		var buf bytes.Buffer
		_ = png.Encode(&buf, imgpkg.CreateBlankImage())
		data, ct = buf.Bytes(), "image/png"
	}

	serveBytes(w, r, data, ct, lastMod, cfg)
}

func serveBytes(w http.ResponseWriter, r *http.Request, body []byte, contentType string, lastMod time.Time, cfg *Config) {
	w.Header().Set("Vary", "Accept")

	etag := makeETag(body)
	if cfg.UseETag {
		if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
			w.Header().Set("ETag", etag)
			setCacheHeaders(w, cfg)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
	}

	w.Header().Set("Content-Type", contentType)
	if !lastMod.IsZero() {
		w.Header().Set("Last-Modified", lastMod.UTC().Format(http.TimeFormat))
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	setCacheHeaders(w, cfg)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func pickFormatByAccept(accept string) string {
	accept = strings.ToLower(accept)
	// AVIF has better compression, prioritize it
	if strings.Contains(accept, "image/avif") {
		return "avif"
	}
	if strings.Contains(accept, "image/webp") {
		return "webp"
	}
	return "png"
}

func makeETag(b []byte) string {
	s := sha256.Sum256(b)
	return "\"" + hex.EncodeToString(s[:16]) + "\""
}

func setCacheHeaders(w http.ResponseWriter, cfg *Config) {
	bsec := int(cfg.BrowserMaxAge.Seconds())
	csec := int(cfg.CDNSMaxAge.Seconds())
	if bsec <= 0 {
		bsec = 86400
	}
	if csec <= 0 {
		csec = bsec
	}
	cc := "public, max-age=" + strconv.Itoa(bsec) + ", s-maxage=" + strconv.Itoa(csec) + ", immutable"
	w.Header().Set("Cache-Control", cc)
	w.Header().Set("Surrogate-Control", "max-age="+strconv.Itoa(csec))
	w.Header().Set("Expires", time.Now().Add(time.Duration(bsec)*time.Second).UTC().Format(http.TimeFormat))
}

func fetchURLCachedWithRevalidation(ctx context.Context, rawURL string, cfg *Config) ([]byte, string, error) {
	canon := discovery.CanonicalizeURLString(rawURL)
	cm := cfg.CacheManager

	// Check cache first (fast path)
	if b, ok := cm.ReadOrigFromCache(canon); ok {
		m, _ := cm.ReadOrigMeta(canon)
		if m.ETag != "" || m.LastModified != "" {
			nb, ct, status, etag, lm, err := fetch.FetchURLConditional(ctx, canon, m.ETag, m.LastModified)
			if err == nil && status == 304 {
				_ = cm.TouchOrigCache(canon)
				_ = cm.WriteOrigMeta(canon, cache.OrigMeta{URL: canon, ETag: m.ETag, LastModified: m.LastModified, UpdatedAt: time.Now()})
				return b, ct, nil
			}
			if err == nil && status == 200 && len(nb) > 0 {
				_ = cm.WriteOrigToCache(canon, nb)
				_ = cm.WriteOrigMeta(canon, cache.OrigMeta{URL: canon, ETag: etag, LastModified: lm, UpdatedAt: time.Now()})
				return nb, ct, nil
			}
			return b, http.DetectContentType(peek512(b)), nil
		}
		return b, http.DetectContentType(peek512(b)), nil
	}

	// Cache miss - use singleflight to prevent thundering herd
	data, err := cfg.fetchGroup.Do(canon, func() ([]byte, error) {
		// Double-check cache in case another goroutine filled it
		if b, ok := cm.ReadOrigFromCache(canon); ok {
			return b, nil
		}

		// Fetch from origin
		b, ct, etag, lm, err := fetch.FetchURLFull(ctx, canon)
		if err != nil {
			return nil, err
		}

		// Store in cache
		_ = cm.WriteOrigToCache(canon, b)
		_ = cm.WriteOrigMeta(canon, cache.OrigMeta{
			URL:          canon,
			ETag:         etag,
			LastModified: lm,
			UpdatedAt:    time.Now(),
		})

		// Store content type in a thread-safe way
		// We'll detect it again after returning from singleflight
		_ = ct // Suppress unused warning
		return b, nil
	})

	if err != nil {
		return nil, "", err
	}

	ct := http.DetectContentType(peek512(data))
	return data, ct, nil
}

func peek512(b []byte) []byte {
	if len(b) > 512 {
		return b[:512]
	}
	return b
}

// CanonicalizeURLString is exported for discovery
func CanonicalizeURLString(raw string) string {
	return discovery.CanonicalizeURLString(raw)
}
