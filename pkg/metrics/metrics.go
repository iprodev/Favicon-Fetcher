package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all application metrics
type Metrics struct {
	// Request metrics
	requestsTotal       uint64
	requestsDuration    sync.Map // URL path -> []float64
	requestsInFlight    int64
	requestsByStatus    sync.Map // Status code -> count
	
	// Cache metrics
	cacheHits           uint64
	cacheMisses         uint64
	cacheSize           int64
	cacheEvictions      uint64
	
	// Error metrics
	errorsTotal         uint64
	errorsByType        sync.Map // Error type -> count
	
	// Icon fetch metrics
	iconFetchesTotal    uint64
	iconFetchDuration   sync.Map // Domain -> []float64
	iconFetchErrors     uint64
	
	// Discovery metrics
	candidatesFound     uint64
	candidatesProcessed uint64
	
	mu sync.RWMutex
}

var (
	globalMetrics = &Metrics{}
	startTime     = time.Now()
)

// Get returns the global metrics instance
func Get() *Metrics {
	return globalMetrics
}

// Reset resets all metrics (for testing)
func Reset() {
	globalMetrics = &Metrics{}
	startTime = time.Now()
}

// Request metrics

func (m *Metrics) IncRequests() {
	atomic.AddUint64(&m.requestsTotal, 1)
}

func (m *Metrics) IncRequestInFlight() {
	atomic.AddInt64(&m.requestsInFlight, 1)
}

func (m *Metrics) DecRequestInFlight() {
	atomic.AddInt64(&m.requestsInFlight, -1)
}

func (m *Metrics) GetRequestsInFlight() int64 {
	return atomic.LoadInt64(&m.requestsInFlight)
}

func (m *Metrics) RecordRequestDuration(path string, duration time.Duration) {
	ms := float64(duration) / float64(time.Millisecond)
	
	val, _ := m.requestsDuration.LoadOrStore(path, &sync.Map{})
	durMap := val.(*sync.Map)
	
	bucket := getBucket(ms)
	count, _ := durMap.LoadOrStore(bucket, new(uint64))
	atomic.AddUint64(count.(*uint64), 1)
}

func (m *Metrics) RecordRequestStatus(status int) {
	count, _ := m.requestsByStatus.LoadOrStore(status, new(uint64))
	atomic.AddUint64(count.(*uint64), 1)
}

// Cache metrics

func (m *Metrics) IncCacheHit() {
	atomic.AddUint64(&m.cacheHits, 1)
}

func (m *Metrics) IncCacheMiss() {
	atomic.AddUint64(&m.cacheMisses, 1)
}

func (m *Metrics) SetCacheSize(size int64) {
	atomic.StoreInt64(&m.cacheSize, size)
}

func (m *Metrics) IncCacheEviction() {
	atomic.AddUint64(&m.cacheEvictions, 1)
}

func (m *Metrics) GetCacheHitRate() float64 {
	hits := atomic.LoadUint64(&m.cacheHits)
	misses := atomic.LoadUint64(&m.cacheMisses)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// Error metrics

func (m *Metrics) IncError(errorType string) {
	atomic.AddUint64(&m.errorsTotal, 1)
	count, _ := m.errorsByType.LoadOrStore(errorType, new(uint64))
	atomic.AddUint64(count.(*uint64), 1)
}

// Icon fetch metrics

func (m *Metrics) IncIconFetch() {
	atomic.AddUint64(&m.iconFetchesTotal, 1)
}

func (m *Metrics) IncIconFetchError() {
	atomic.AddUint64(&m.iconFetchErrors, 1)
}

func (m *Metrics) RecordIconFetchDuration(domain string, duration time.Duration) {
	ms := float64(duration) / float64(time.Millisecond)
	
	val, _ := m.iconFetchDuration.LoadOrStore(domain, &sync.Map{})
	durMap := val.(*sync.Map)
	
	bucket := getBucket(ms)
	count, _ := durMap.LoadOrStore(bucket, new(uint64))
	atomic.AddUint64(count.(*uint64), 1)
}

// Discovery metrics

func (m *Metrics) AddCandidatesFound(count int) {
	atomic.AddUint64(&m.candidatesFound, uint64(count))
}

func (m *Metrics) AddCandidatesProcessed(count int) {
	atomic.AddUint64(&m.candidatesProcessed, uint64(count))
}

// Prometheus exposition

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		
		// General info
		writeMetric(w, "favicon_build_info", "gauge", 1, map[string]string{
			"version": "1.0.0",
		})
		writeMetric(w, "favicon_uptime_seconds", "gauge", time.Since(startTime).Seconds(), nil)
		
		// Request metrics
		writeMetric(w, "favicon_requests_total", "counter", atomic.LoadUint64(&m.requestsTotal), nil)
		writeMetric(w, "favicon_requests_in_flight", "gauge", m.GetRequestsInFlight(), nil)
		
		// Write request duration histogram
		m.requestsDuration.Range(func(key, value interface{}) bool {
			path := key.(string)
			durMap := value.(*sync.Map)
			durMap.Range(func(k, v interface{}) bool {
				bucket := k.(string)
				count := atomic.LoadUint64(v.(*uint64))
				writeMetric(w, "favicon_request_duration_milliseconds_bucket", "counter", count, map[string]string{
					"path": path,
					"le":   bucket,
				})
				return true
			})
			return true
		})
		
		// Write status code metrics
		m.requestsByStatus.Range(func(key, value interface{}) bool {
			status := key.(int)
			count := atomic.LoadUint64(value.(*uint64))
			writeMetric(w, "favicon_requests_by_status_total", "counter", count, map[string]string{
				"status": http.StatusText(status),
				"code":   fmt.Sprintf("%d", status),
			})
			return true
		})
		
		// Cache metrics
		writeMetric(w, "favicon_cache_hits_total", "counter", atomic.LoadUint64(&m.cacheHits), nil)
		writeMetric(w, "favicon_cache_misses_total", "counter", atomic.LoadUint64(&m.cacheMisses), nil)
		writeMetric(w, "favicon_cache_hit_rate", "gauge", m.GetCacheHitRate(), nil)
		writeMetric(w, "favicon_cache_size_bytes", "gauge", atomic.LoadInt64(&m.cacheSize), nil)
		writeMetric(w, "favicon_cache_evictions_total", "counter", atomic.LoadUint64(&m.cacheEvictions), nil)
		
		// Error metrics
		writeMetric(w, "favicon_errors_total", "counter", atomic.LoadUint64(&m.errorsTotal), nil)
		m.errorsByType.Range(func(key, value interface{}) bool {
			errorType := key.(string)
			count := atomic.LoadUint64(value.(*uint64))
			writeMetric(w, "favicon_errors_by_type_total", "counter", count, map[string]string{
				"type": errorType,
			})
			return true
		})
		
		// Icon fetch metrics
		writeMetric(w, "favicon_icon_fetches_total", "counter", atomic.LoadUint64(&m.iconFetchesTotal), nil)
		writeMetric(w, "favicon_icon_fetch_errors_total", "counter", atomic.LoadUint64(&m.iconFetchErrors), nil)
		
		// Discovery metrics
		writeMetric(w, "favicon_candidates_found_total", "counter", atomic.LoadUint64(&m.candidatesFound), nil)
		writeMetric(w, "favicon_candidates_processed_total", "counter", atomic.LoadUint64(&m.candidatesProcessed), nil)
	}
}

func writeMetric(w http.ResponseWriter, name, metricType string, value interface{}, labels map[string]string) {
	// Write TYPE comment (once per metric name)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
	
	// Write metric
	fmt.Fprint(w, name)
	
	if len(labels) > 0 {
		fmt.Fprint(w, "{")
		first := true
		for k, v := range labels {
			if !first {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, "%s=\"%s\"", k, v)
			first = false
		}
		fmt.Fprint(w, "}")
	}
	
	fmt.Fprint(w, " ")
	
	switch v := value.(type) {
	case int:
		fmt.Fprintf(w, "%d", v)
	case int64:
		fmt.Fprintf(w, "%d", v)
	case uint64:
		fmt.Fprintf(w, "%d", v)
	case float64:
		fmt.Fprintf(w, "%.6f", v)
	}
	
	fmt.Fprint(w, "\n")
}

func getBucket(ms float64) string {
	buckets := []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}
	for _, b := range buckets {
		if ms <= b {
			return fmt.Sprintf("%.0f", b)
		}
	}
	return "+Inf"
}

// Middleware for automatic request tracking
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := Get()
		m.IncRequests()
		m.IncRequestInFlight()
		defer m.DecRequestInFlight()
		
		start := time.Now()
		
		// Wrap response writer to capture status
		sw := &statusWriter{ResponseWriter: w, status: 200}
		
		next.ServeHTTP(sw, r)
		
		duration := time.Since(start)
		m.RecordRequestDuration(r.URL.Path, duration)
		m.RecordRequestStatus(sw.status)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
