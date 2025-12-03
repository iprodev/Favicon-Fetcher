# Favicon Fetcher API Documentation

## Overview

Favicon Fetcher is a high-performance HTTP service for fetching, caching, and serving website favicons. It automatically discovers favicons from web pages, handles multiple formats, and provides intelligent caching.

## Base URL

```
http://localhost:9090
```

## Endpoints

### GET /favicons

Fetch and serve a favicon for a given URL or domain.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | Yes* | - | Full URL of the website (e.g., `https://example.com`) |
| `domain` | string | Yes* | - | Domain name (e.g., `example.com`) - automatically adds https:// |
| `sz` or `size` | integer | No | 32 | Output size in pixels (min: 16, max: 256) |

*Either `url` or `domain` must be provided

#### Headers

| Header | Description |
|--------|-------------|
| `Accept` | Specify preferred format. Supports `image/avif`, `image/webp`, and `image/png` |
| `If-None-Match` | ETag for conditional requests (304 responses) |

#### Response

**Success (200 OK)**

Returns the favicon image in PNG or WebP format.

Headers:
- `Content-Type`: `image/png` or `image/webp`
- `Cache-Control`: Public cache directives
- `ETag`: Entity tag for caching
- `Last-Modified`: Last modification time
- `Expires`: Cache expiration time

**Not Modified (304)**

Returned when the `If-None-Match` header matches the current ETag.

**Examples**

```bash
# Fetch favicon by URL
curl "http://localhost:9090/favicons?url=https://dignitydash.com"

# Fetch favicon by domain
curl "http://localhost:9090/favicons?domain=dignitydash.com"

# Request specific size
curl "http://localhost:9090/favicons?url=https://dignitydash.com&sz=64"

# Request WebP format
curl -H "Accept: image/webp" "http://localhost:9090/favicons?url=https://dignitydash.com"

# Request AVIF format (best compression)
curl -H "Accept: image/avif" "http://localhost:9090/favicons?url=https://dignitydash.com"

# Conditional request with ETag
curl -H "If-None-Match: \"abc123\"" "http://localhost:9090/favicons?url=https://dignitydash.com"
```

### GET /health

Health check endpoint.

#### Response

**Success (200 OK)**

```json
{
  "status": "ok"
}
```

## Features

### Icon Discovery

The service automatically discovers favicons through multiple methods:

1. **HTML parsing**: Searches for `<link rel="icon">`, `<link rel="apple-touch-icon">`, and shortcut icons
2. **Root fallback**: Tries `/favicon.ico` at the domain root
3. **Format prioritization**: Prefers SVG → PNG/ICO → other formats
4. **Size matching**: Selects the icon closest to the requested size

### Supported Formats

**Input formats:**
- ICO (with multi-resolution support)
- SVG (rasterized to requested size)
- PNG
- JPEG
- GIF
- WebP
- AVIF
- BMP

**Output formats:**
- PNG (default)
- WebP (when requested via Accept header)
- AVIF (when requested via Accept header, best compression)

### Caching

**Three-tier cache system:**

1. **Original cache**: Stores raw downloaded icons
2. **Resized cache**: Stores processed/resized versions
3. **Fallback cache**: Default globe icon

**Cache features:**
- Configurable TTL (default: 24 hours)
- HTTP conditional requests (ETag, Last-Modified)
- Automatic cleanup (janitor process)
- Size-based eviction
- Atomic writes for consistency

### Security

**Built-in protections:**
- Blocks private IP ranges (RFC 1918)
- Blocks localhost and loopback addresses
- DNS validation before requests
- Scheme validation (HTTP/HTTPS only)
- Redirect limits (max 8)
- Size limits (4MB for images, 1MB for HTML)
- Request timeouts (12 seconds)

## Configuration

### Command-Line Flags

```bash
./server [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-addr` | string | - | Listen address (e.g., `:9090`, `0.0.0.0:8080`) |
| `-port` | int | - | Port number (alternative to `-addr`) |
| `-cache-dir` | string | `./cache` | Directory for cache storage |
| `-cache-ttl` | duration | `24h` | Time-to-live for cache entries |
| `-browser-max-age` | duration | `cache-ttl` | Browser cache duration (Cache-Control: max-age) |
| `-cdn-smax-age` | duration | `browser-max-age` | CDN cache duration (Cache-Control: s-maxage) |
| `-etag` | bool | `true` | Enable ETag support |
| `-janitor-interval` | duration | `30m` | Cache cleanup interval (0 to disable) |
| `-max-cache-size-bytes` | int64 | `0` | Maximum cache size in bytes (0 = unlimited) |
| `-log-level` | string | `info` | Log level (debug, info, warn, error) |
| `-help` | bool | `false` | Show help and exit |

### Environment Variables

- `PORT`: Alternative to `-port` flag

### Examples

```bash
# Basic usage
./server

# Custom port
./server -port 8080

# Custom cache directory and TTL
./server -cache-dir /var/cache/favicons -cache-ttl 48h

# Disable janitor
./server -janitor-interval 0

# Maximum cache size (1GB)
./server -max-cache-size-bytes 1073741824

# Debug logging
./server -log-level debug

# Production configuration
./server -addr :80 \
  -cache-dir /var/cache/favicons \
  -cache-ttl 72h \
  -browser-max-age 24h \
  -cdn-smax-age 72h \
  -max-cache-size-bytes 10737418240 \
  -log-level info
```

## Building

### Build

```bash
# Full format support (PNG, WebP, AVIF)
go build -o server ./cmd/server
```

All formats are included by default. No build tags needed.

## Error Handling

### Fallback Behavior

When a favicon cannot be fetched or processed:
1. Returns a default globe icon
2. HTTP 200 status (not 404)
3. Proper caching headers

This ensures the service never fails completely and provides a consistent user experience.

### Blocked URLs

Requests to blocked URLs (localhost, private IPs, etc.) return the fallback icon with HTTP 200.

## Performance

### Recommendations

- Use a CDN or reverse proxy (nginx, Cloudflare) in front of the service
- Enable AVIF for modern browsers (40-60% smaller files, best compression)
- Enable WebP for broader browser support (30-50% smaller files)
- Set appropriate cache TTLs based on your use case
- Monitor cache size and adjust limits as needed
- Use the janitor to prevent unbounded cache growth

### Monitoring

Check the `/health` endpoint for availability:

```bash
curl http://localhost:9090/health
```

Review logs for:
- Request patterns and latency
- Cache hit/miss rates
- Janitor cleanup activity
- Failed fetches and errors

## Common Use Cases

### Website Favicon Display

```html
<img src="http://localhost:9090/favicons?url=https://example.com" 
     alt="favicon" 
     width="32" 
     height="32">
```

### Bookmark Manager

```javascript
fetch(`http://localhost:9090/favicons?domain=${domain}&sz=64`)
  .then(response => response.blob())
  .then(blob => {
    const img = document.createElement('img');
    img.src = URL.createObjectURL(blob);
    document.body.appendChild(img);
  });
```

### Browser Extension

```javascript
chrome.bookmarks.getRecent(10, (bookmarks) => {
  bookmarks.forEach(bookmark => {
    const faviconUrl = `http://localhost:9090/favicons?url=${bookmark.url}&sz=16`;
    // Display favicon
  });
});
```

## Troubleshooting

### No favicon returned

1. Check if the website has a favicon
2. Verify the URL is accessible (not blocked by firewall)
3. Check logs for DNS resolution issues
4. Ensure the service has internet access

### Cache not working

1. Verify cache directory permissions
2. Check available disk space
3. Review cache TTL settings
4. Ensure janitor isn't too aggressive

### High memory usage

1. Reduce `-max-cache-size-bytes`
2. Decrease `-cache-ttl`
3. Lower `-janitor-interval` for more frequent cleanup
4. Monitor for large favicon files

## License

[Add your license information here]

## Support

[Add your support/contact information here]
