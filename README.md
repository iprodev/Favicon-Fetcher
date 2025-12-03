# Favicon Fetcher

A high-performance, production-ready HTTP service for fetching, caching, and serving website favicons.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/iprodev/Favicon-Fetcher/actions/workflows/ci.yml/badge.svg)](https://github.com/iprodev/Favicon-Fetcher/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/iprodev/Favicon-Fetcher)](https://github.com/iprodev/Favicon-Fetcher/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Smart Discovery** - Automatically finds favicons from HTML `<link>` tags, Apple Touch Icons, and `/favicon.ico` fallback
- **Multi-Format Support** - Reads ICO, SVG, PNG, JPEG, GIF, WebP, AVIF, BMP
- **Modern Output Formats** - Serves PNG, WebP, or AVIF based on `Accept` header
- **High-Quality SVG Rendering** - Uses [tdewolff/canvas](https://github.com/tdewolff/canvas) for accurate SVG rasterization
- **3-Tier Caching** - Original images, resized versions, and fallback icons with configurable TTL
- **Security First** - SSRF protection, private IP blocking, DNS rebinding prevention
- **Production Ready** - Rate limiting, Prometheus metrics, graceful shutdown, Docker support
- **Request Deduplication** - Singleflight pattern prevents thundering herd

## Quick Start

### Using Docker

```bash
docker run -d -p 9090:9090 ghcr.io/iprodev/favicon-fetcher:latest
```

### Using Docker Compose

```bash
curl -O https://raw.githubusercontent.com/iprodev/Favicon-Fetcher/main/docker-compose.yml
docker-compose up -d
```

### Download Binary

Download the latest release from [Releases](https://github.com/iprodev/Favicon-Fetcher/releases):

```bash
# Linux (amd64)
curl -LO https://github.com/iprodev/Favicon-Fetcher/releases/latest/download/favicon-server-linux-amd64
chmod +x favicon-server-linux-amd64
./favicon-server-linux-amd64

# macOS (Apple Silicon)
curl -LO https://github.com/iprodev/Favicon-Fetcher/releases/latest/download/favicon-server-darwin-arm64
chmod +x favicon-server-darwin-arm64
./favicon-server-darwin-arm64
```

### Build from Source

```bash
git clone https://github.com/iprodev/Favicon-Fetcher.git
cd Favicon-Fetcher
go build -o favicon-server ./cmd/server
./favicon-server
```

## Usage

### Basic Usage

```bash
# Fetch favicon by URL
curl "http://localhost:9090/favicons?url=https://dignitydash.com" -o favicon.png

# Fetch by domain
curl "http://localhost:9090/favicons?domain=dignitydash.com" -o favicon.png

# Specify size (16-256 pixels)
curl "http://localhost:9090/favicons?url=https://dignitydash.com&sz=64" -o favicon.png

# Request WebP format
curl -H "Accept: image/webp" "http://localhost:9090/favicons?url=https://dignitydash.com" -o favicon.webp

# Request AVIF format (best compression)
curl -H "Accept: image/avif" "http://localhost:9090/favicons?url=https://dignitydash.com" -o favicon.avif
```

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /favicons` | Fetch and serve favicon |
| `GET /health` | Health check |
| `GET /metrics` | Prometheus metrics |

### Query Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `url` | - | Full URL (e.g., `https://example.com/page`) |
| `domain` | - | Domain only (e.g., `example.com`) |
| `sz` or `size` | 32 | Output size in pixels (16-256) |

## Configuration

### Command-Line Flags

```bash
./favicon-server [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:9090` | Listen address |
| `-port` | `9090` | Port number |
| `-cache-dir` | `./cache` | Cache directory |
| `-cache-ttl` | `24h` | Cache TTL |
| `-browser-max-age` | `=cache-ttl` | Browser cache duration |
| `-cdn-smax-age` | `=browser-max-age` | CDN cache duration |
| `-etag` | `true` | Enable ETag support |
| `-janitor-interval` | `30m` | Cache cleanup interval |
| `-max-cache-size-bytes` | `0` | Max cache size (0=unlimited) |
| `-rate-limit` | `0` | Global requests/sec (0=unlimited) |
| `-ip-rate-limit` | `0` | Per-IP requests/sec (0=unlimited) |
| `-log-level` | `info` | Log level (debug/info/warn/error) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Alternative to `-port` flag |

### Example Configurations

**Development:**
```bash
./favicon-server -log-level debug -cache-ttl 1h
```

**Production:**
```bash
./favicon-server \
  -addr :80 \
  -cache-dir /var/cache/favicons \
  -cache-ttl 72h \
  -browser-max-age 24h \
  -cdn-smax-age 72h \
  -max-cache-size-bytes 10737418240 \
  -rate-limit 1000 \
  -ip-rate-limit 100 \
  -log-level info
```

## Docker

### Docker Compose

```yaml
version: '3.8'
services:
  favicon-server:
    image: ghcr.io/iprodev/favicon-fetcher:latest
    ports:
      - "9090:9090"
    volumes:
      - favicon-cache:/app/cache
    environment:
      - PORT=9090
    restart: unless-stopped

volumes:
  favicon-cache:
```

### Build Custom Image

```bash
docker build -t favicon-fetcher .
docker run -p 9090:9090 favicon-fetcher
```

## Output Formats

The service negotiates output format based on the `Accept` header:

| Accept Header | Output | Compression |
|---------------|--------|-------------|
| `image/avif` | AVIF | Best (35-60% of PNG) |
| `image/webp` | WebP | Good (50-70% of PNG) |
| `*/*` (default) | PNG | Baseline |

## Security

Built-in protections:

- **SSRF Protection** - Blocks private IPs (10.x, 172.16.x, 192.168.x), localhost, loopback
- **DNS Rebinding Prevention** - Validates resolved IPs before connection
- **Scheme Validation** - Only HTTP/HTTPS allowed
- **Size Limits** - 4MB for images, 1MB for HTML
- **Redirect Limits** - Maximum 8 redirects
- **Request Timeout** - 12 seconds

## Monitoring

### Health Check

```bash
curl http://localhost:9090/health
# {"status":"ok"}
```

### Prometheus Metrics

```bash
curl http://localhost:9090/metrics
```

Available metrics:
- `favicon_requests_total` - Total requests
- `favicon_requests_in_flight` - Current active requests
- `favicon_cache_hits_total` / `favicon_cache_misses_total` - Cache statistics
- `favicon_cache_hit_rate` - Cache hit ratio
- `favicon_errors_total` - Error count by type

## Architecture

```
Favicon-Fetcher/
├── cmd/server/          # Application entry point
├── internal/
│   ├── cache/          # 3-tier caching system
│   ├── discovery/      # Favicon discovery from HTML
│   ├── fetch/          # HTTP client with security
│   ├── handler/        # HTTP handlers
│   ├── image/          # Image processing (decode/encode/resize)
│   └── security/       # SSRF protection, IP validation
├── pkg/
│   ├── logger/         # Structured logging
│   ├── metrics/        # Prometheus metrics
│   └── ratelimit/      # Token bucket rate limiter
└── tests/              # Integration tests
```

## Development

### Prerequisites

- Go 1.22+
- Make (optional)

### Build

```bash
make build
# or
go build -o favicon-server ./cmd/server
```

### Test

```bash
make test
# or
go test ./...
```

### Lint

```bash
make lint
# requires golangci-lint
```

## Changelog

### v1.0.0

- Initial release
- Multi-format support: ICO, SVG, PNG, JPEG, GIF, WebP, AVIF, BMP
- Output formats: PNG, WebP, AVIF
- High-quality SVG rendering with tdewolff/canvas
- 3-tier caching system
- SSRF protection and security features
- Rate limiting (global and per-IP)
- Prometheus metrics
- Docker support with multi-arch images
- GitHub Actions CI/CD

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [tdewolff/canvas](https://github.com/tdewolff/canvas) - SVG rendering
- [go-ico](https://github.com/sergeymakinen/go-ico) - ICO decoding
- [go-webp](https://github.com/kolesa-team/go-webp) - WebP encoding
- [gen2brain/avif](https://github.com/gen2brain/avif) - AVIF encoding
