// Package security provides security validations and protections against
// common web vulnerabilities including SSRF, DNS rebinding, and private IP access.
package security

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

var blockedNets []*net.IPNet

func init() {
	// Block private ranges
	for _, cidr := range []string{
		"127.0.0.0/8", "::1/128",
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"169.254.0.0/16", "100.64.0.0/10",
		"0.0.0.0/8", "224.0.0.0/4", "240.0.0.0/4",
		"::/128", "fe80::/10", "fc00::/7", "ff00::/8",
	} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			blockedNets = append(blockedNets, n)
		}
	}
}

// IsBlockedIP checks if an IP address is in a blocked network range.
// Blocked ranges include private IPs (RFC 1918), localhost, link-local,
// and other reserved ranges.
func IsBlockedIP(ip net.IP) bool {
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// IsAllowedScheme checks if a URL uses an allowed scheme.
// Only HTTP and HTTPS schemes are permitted.
func IsAllowedScheme(u *url.URL) bool {
	return u != nil && (u.Scheme == "http" || u.Scheme == "https")
}

// NormalizeURL parses and validates a URL string, adding https:// if no scheme is present.
// It performs multiple security checks:
//   - Validates the URL format
//   - Checks for empty hostname
//   - Validates scheme (HTTP/HTTPS only)
//   - Blocks localhost
//   - Blocks private IP addresses
//   - Performs DNS resolution and validates resolved IPs
//
// Returns the parsed URL and nil if valid, or nil and an error if validation fails.
func NormalizeURL(in string) (*url.URL, error) {
	if !strings.Contains(in, "://") {
		in = "https://" + in
	}
	u, err := url.Parse(in)
	if err != nil {
		return nil, err
	}
	if u.Hostname() == "" {
		return nil, errors.New("empty hostname")
	}
	if !IsAllowedScheme(u) {
		return nil, errors.New("only http/https allowed")
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return nil, errors.New("localhost not allowed")
	}

	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return nil, errors.New("private ip not allowed")
		}
		return u, nil
	}

	if !strings.Contains(host, ".") {
		return nil, errors.New("hostname must contain a dot")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("hostname not resolvable")
	}

	for _, ipa := range ips {
		if !IsBlockedIP(ipa.IP) {
			return u, nil
		}
	}
	return nil, errors.New("hostname resolves to private range only")
}

// ValidatedDialContext performs DNS resolution and validates IPs before connecting.
// This prevents DNS rebinding attacks by resolving and validating in a single atomic operation.
//
// The function:
//   - Validates IP addresses immediately after resolution
//   - Uses a short DNS lookup timeout to prevent timing attacks
//   - Connects directly to the validated IP to bypass subsequent DNS lookups
//   - Filters out all blocked IP addresses
//
// Returns a network connection or an error if all resolved IPs are blocked.
func ValidatedDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: 7 * time.Second,
		// Force a fresh DNS lookup every time to prevent caching issues
		Resolver: &net.Resolver{
			PreferGo: true,
		},
	}

	// If host is already an IP address, validate it directly
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return nil, errors.New("blocked ip")
		}
		return dialer.DialContext(ctx, network, address)
	}

	// Resolve hostname to IPs
	// Using a short timeout to prevent DNS rebinding timing attacks
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("hostname did not resolve to any ips")
	}

	// Validate all resolved IPs before attempting connection
	// This prevents connecting even if first IP is blocked
	var allowedIP net.IP
	for _, ipa := range ips {
		if !IsBlockedIP(ipa.IP) {
			allowedIP = ipa.IP
			break
		}
	}

	if allowedIP == nil {
		return nil, errors.New("all resolved ips are blocked")
	}

	// Connect directly to the validated IP to prevent DNS rebinding
	// This bypasses any subsequent DNS lookups
	return dialer.DialContext(ctx, network, net.JoinHostPort(allowedIP.String(), port))
}
