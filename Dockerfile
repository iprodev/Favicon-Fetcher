# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Enable auto toolchain to download required Go version
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Build the application with full format support (PNG, WebP, AVIF)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o favicon-server \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 favicon && \
    adduser -D -u 1000 -G favicon favicon

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/favicon-server /app/favicon-server

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create cache directory
RUN mkdir -p /app/cache && \
    chown -R favicon:favicon /app

# Switch to non-root user
USER favicon

# Expose port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:9090/health || exit 1

# Set environment variables
ENV PORT=9090 \
    CACHE_DIR=/app/cache

# Run the application
ENTRYPOINT ["/app/favicon-server"]
CMD ["-addr", ":9090", "-cache-dir", "/app/cache", "-log-level", "info"]
