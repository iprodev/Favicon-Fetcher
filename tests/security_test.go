package tests

import (
	"net"
	"testing"

	"faviconsvc/internal/security"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := security.IsBlockedIP(ip)
			if result != tt.blocked {
				t.Errorf("IsBlockedIP(%s) = %v, want %v", tt.ip, result, tt.blocked)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"example.com", false},
		{"https://example.com", false},
		{"http://example.com", false},
		{"localhost", true},
		{"http://localhost", true},
		{"http://127.0.0.1", true},
		{"http://10.0.0.1", true},
		{"ftp://example.com", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := security.NormalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeURL(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func parseIP(s string) net.IP {
	return net.ParseIP(s)
}
