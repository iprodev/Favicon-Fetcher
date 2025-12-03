package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_Unlimited(t *testing.T) {
	tests := []struct {
		name            string
		globalRate      int
		globalBurst     int
		ipRate          int
		ipBurst         int
		expectLimiter   bool
		testRequests    int
		expectAllAllowed bool
	}{
		{
			name:             "Both unlimited (0,0)",
			globalRate:       0,
			globalBurst:      0,
			ipRate:           0,
			ipBurst:          0,
			expectLimiter:    false,
			testRequests:     100,
			expectAllAllowed: true,
		},
		{
			name:             "IP unlimited, global limited",
			globalRate:       10,
			globalBurst:      20,
			ipRate:           0,
			ipBurst:          0,
			expectLimiter:    true,
			testRequests:     30,
			expectAllAllowed: false,
		},
		{
			name:             "Global unlimited, IP limited",
			globalRate:       0,
			globalBurst:      0,
			ipRate:           5,
			ipBurst:          10,
			expectLimiter:    true,
			testRequests:     20,
			expectAllAllowed: false,
		},
		{
			name:             "Both limited",
			globalRate:       100,
			globalBurst:      200,
			ipRate:           10,
			ipBurst:          20,
			expectLimiter:    true,
			testRequests:     30,
			expectAllAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create limiter (or not)
			var limiter *Limiter
			if tt.globalRate > 0 || tt.ipRate > 0 {
				limiter = NewLimiter(tt.globalRate, tt.globalBurst, tt.ipRate, tt.ipBurst)
				defer limiter.Stop()
			}

			// Check if limiter was created as expected
			if (limiter != nil) != tt.expectLimiter {
				t.Errorf("Expected limiter=%v, got limiter=%v", tt.expectLimiter, limiter != nil)
			}

			// If no limiter expected, skip request testing
			if limiter == nil {
				return
			}

			// Test requests
			allowed := 0
			denied := 0
			testIP := "192.168.1.1"

			for i := 0; i < tt.testRequests; i++ {
				if limiter.Allow(testIP) {
					allowed++
				} else {
					denied++
				}
			}

			// Check if all were allowed as expected
			if tt.expectAllAllowed {
				if denied > 0 {
					t.Errorf("Expected all %d requests to be allowed, but %d were denied", tt.testRequests, denied)
				}
			} else {
				if denied == 0 {
					t.Errorf("Expected some requests to be denied, but all %d were allowed", tt.testRequests)
				}
			}

			t.Logf("Allowed: %d, Denied: %d", allowed, denied)
		})
	}
}

func TestLimiter_IPUnlimited(t *testing.T) {
	// Create limiter with IP rate = 0 (unlimited)
	limiter := NewLimiter(0, 0, 0, 0)
	if limiter != nil {
		t.Error("Expected nil limiter when both rates are 0")
		limiter.Stop()
		return
	}

	// Create limiter with only IP rate = 0
	limiter = NewLimiter(100, 200, 0, 0)
	defer limiter.Stop()

	// Test that IP limiting is disabled
	testIP := "10.0.0.1"
	allowed := 0

	// Try 1000 requests - should not be limited by IP
	for i := 0; i < 1000; i++ {
		if limiter.Allow(testIP) {
			allowed++
		}
		// Small delay to not hit global limit instantly
		time.Sleep(time.Millisecond)
	}

	// With global rate of 100/s and 1000 requests over 1 second,
	// most should be allowed (burst helps)
	if allowed < 100 {
		t.Errorf("Expected at least 100 requests allowed with global rate 100/s, got %d", allowed)
	}

	t.Logf("With ipRate=0 (unlimited): %d/%d requests allowed", allowed, 1000)
}

func TestLimiter_GlobalUnlimited(t *testing.T) {
	// Create limiter with global rate = 0 (unlimited)
	limiter := NewLimiter(0, 0, 5, 10)
	defer limiter.Stop()

	// Test multiple IPs
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, ip := range ips {
		allowed := 0
		denied := 0

		// Try 20 requests per IP
		for i := 0; i < 20; i++ {
			if limiter.Allow(ip) {
				allowed++
			} else {
				denied++
			}
		}

		// With IP rate of 5/s and burst of 10, should allow ~10-15 initially
		if allowed < 5 {
			t.Errorf("IP %s: Expected at least 5 allowed, got %d", ip, allowed)
		}

		t.Logf("IP %s: Allowed=%d, Denied=%d", ip, allowed, denied)
	}
}

func TestTokenBucket_ZeroRate(t *testing.T) {
	// This shouldn't happen in practice due to checks in Allow(),
	// but let's ensure it doesn't panic
	bucket := newTokenBucket(0, 0)

	// Should not panic
	allowed := bucket.allow()

	// With zero rate, should be denied after initial token is used
	t.Logf("Zero rate bucket allowed: %v", allowed)
}
