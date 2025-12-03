# Rate Limiting Examples

This document provides examples of different rate limiting configurations.

## Understanding Rate Limiting

The favicon service supports two types of rate limiting:

1. **Global Rate Limiting**: Limits total requests per second across all clients
2. **Per-IP Rate Limiting**: Limits requests per second from each individual IP address

Both can be used together or separately. Setting a limit to `0` disables that type of rate limiting.

## Configuration Examples

### 1. No Rate Limiting (Default)

**Use case**: Development, testing, or trusted internal networks

```bash
./favicon-server
# Or explicitly:
./favicon-server -rate-limit 0 -ip-rate-limit 0
```

**Behavior**: Unlimited requests from all sources

---

### 2. IP Rate Limiting Only (Recommended for Production)

**Use case**: Public-facing service where you want to prevent individual IP abuse

```bash
# Allow 10 requests/second per IP
./favicon-server -ip-rate-limit 10

# Allow 50 requests/second per IP with custom burst
./favicon-server -ip-rate-limit 50 -ip-rate-limit-burst 100

# High-traffic: 100 requests/second per IP
./favicon-server -ip-rate-limit 100 -ip-rate-limit-burst 200
```

**Behavior**:
- Each IP can make up to N requests per second
- Burst allows temporary spikes (default: 2x the rate)
- Global throughput is unlimited
- Good for preventing individual client abuse

---

### 3. Global Rate Limiting Only

**Use case**: When you have a reverse proxy handling IP-based limiting, or you want to cap total server load

```bash
# Limit server to 1000 requests/second total
./favicon-server -rate-limit 1000 -ip-rate-limit 0

# Limit to 500 requests/second with custom burst
./favicon-server -rate-limit 500 -rate-limit-burst 1000 -ip-rate-limit 0
```

**Behavior**:
- Total server capacity is capped at N req/s
- Individual IPs can use as much as they want (until global limit is hit)
- First come, first served
- Good for capacity planning

---

### 4. Both Global and IP Rate Limiting

**Use case**: Maximum control for public services

```bash
# 10,000 req/s total, 50 req/s per IP
./favicon-server -rate-limit 10000 -ip-rate-limit 50

# With custom burst values
./favicon-server \
  -rate-limit 10000 \
  -rate-limit-burst 20000 \
  -ip-rate-limit 50 \
  -ip-rate-limit-burst 100
```

**Behavior**:
- Provides both per-client fairness AND server protection
- Individual IPs are limited to 50 req/s
- Total server load is capped at 10,000 req/s
- Best for large-scale public deployments

---

### 5. Development/Testing Mode

**Use case**: Local development, integration tests

```bash
# No limits at all
./favicon-server -log-level debug -ip-rate-limit 0
```

**Behavior**: No rate limiting, verbose logging

---

### 6. Production Recommended

**Use case**: General production deployment

```bash
./favicon-server \
  -addr :80 \
  -cache-dir /var/cache/favicons \
  -cache-ttl 72h \
  -ip-rate-limit 10 \
  -ip-rate-limit-burst 20 \
  -max-cache-size-bytes 10737418240 \
  -janitor-interval 30m \
  -log-level info
```

**Behavior**:
- 10 req/s per IP (prevents abuse)
- 20 request burst per IP (handles spikes)
- No global limit (scales with load)
- Appropriate for most use cases

---

## Understanding Burst Capacity

**What is burst?**
- Burst allows temporary rate spikes beyond the sustained rate
- Default: 2x the rate limit
- Example: `rate=10, burst=20` means:
  - Can do 20 requests immediately (burst)
  - Then limited to 10 per second sustained

**When to adjust burst:**
- **Higher burst**: For bursty traffic patterns (web browsers, batch jobs)
- **Lower burst**: For steady traffic, or to prevent stampedes
- **Equal to rate**: No burst allowed (strict rate enforcement)

```bash
# Strict rate (no burst)
./favicon-server -ip-rate-limit 10 -ip-rate-limit-burst 10

# Generous burst (3x)
./favicon-server -ip-rate-limit 10 -ip-rate-limit-burst 30

# Auto burst (2x) - recommended
./favicon-server -ip-rate-limit 10
# burst will be set to 20 automatically
```

---

## Testing Rate Limits

### Test IP Rate Limiting

```bash
# Start server with 5 req/s per IP
./favicon-server -ip-rate-limit 5 -log-level debug

# In another terminal, spam requests
for i in {1..20}; do
  curl -s http://localhost:9090/favicons?domain=github.com > /dev/null
  echo "Request $i: $?"
done

# You should see:
# - First 10 succeed (burst)
# - Then ~5 per second succeed
# - Rest get HTTP 429 (Too Many Requests)
```

### Test Global Rate Limiting

```bash
# Start server with 100 req/s global
./favicon-server -rate-limit 100 -ip-rate-limit 0

# Parallel requests from multiple IPs (simulate load)
ab -n 1000 -c 50 http://localhost:9090/health

# Check metrics
curl http://localhost:9090/metrics | grep rate_limit
```

---

## Monitoring Rate Limits

Check the `/metrics` endpoint for rate limit stats:

```bash
curl http://localhost:9090/metrics | grep -E "favicon_errors.*rate_limit"
```

You'll see:
- `favicon_errors_by_type_total{type="rate_limit_global"}`: Global limit hits
- `favicon_errors_by_type_total{type="rate_limit_ip"}`: Per-IP limit hits

---

## Behind a Reverse Proxy

If you're behind nginx/Cloudflare:

### Option 1: Use Both Layers

```bash
# Application: Per-IP limiting
./favicon-server -ip-rate-limit 50

# Nginx: Global limiting + DDoS protection
# (see nginx config in docs/Deployment.md)
```

### Option 2: Proxy-Only

```bash
# Disable app-level rate limiting
./favicon-server -ip-rate-limit 0

# Let nginx/Cloudflare handle all rate limiting
```

**Recommendation**: Use both for defense in depth!

---

## Common Scenarios

### Small Personal Site
```bash
./favicon-server -ip-rate-limit 5
```

### Medium Business Site
```bash
./favicon-server -ip-rate-limit 20
```

### Large Public Service
```bash
./favicon-server -rate-limit 10000 -ip-rate-limit 50
```

### Internal Service (Trusted Network)
```bash
./favicon-server -ip-rate-limit 0
```

### API Service (With API Keys)
```bash
# Generous per-IP, assuming API key auth adds another layer
./favicon-server -ip-rate-limit 100
```

---

## Troubleshooting

### "Getting 429 errors constantly"

**Cause**: Rate limit too low for your traffic

**Solution**: 
```bash
# Check current metrics
curl http://localhost:9090/metrics | grep rate_limit

# Increase limits
./favicon-server -ip-rate-limit 50 -ip-rate-limit-burst 100
```

### "Want to temporarily disable for testing"

```bash
# Disable all rate limiting
./favicon-server -rate-limit 0 -ip-rate-limit 0
```

### "Different limits for different paths?"

Currently not supported. Use nginx for path-based rate limiting:

```nginx
location /favicons {
    limit_req zone=favicons burst=20;
    proxy_pass http://backend;
}
```

---

## Best Practices

1. ✅ **Start conservative**: Begin with `ip-rate-limit 10` and increase if needed
2. ✅ **Monitor metrics**: Watch for rate limit errors in `/metrics`
3. ✅ **Set burst = 2x rate**: Allows natural traffic spikes
4. ✅ **Use IP limiting in production**: Prevents individual abuse
5. ✅ **Global limit for capacity planning**: Cap total server load
6. ✅ **Disable in development**: Use `-ip-rate-limit 0` when testing
7. ✅ **Layer with proxy**: nginx + app = defense in depth
8. ✅ **Test before deploying**: Verify limits work as expected

---

**Need help?** Check the [main README](../README.md) or open an issue!
