package security

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
)

type Resolver func(host string) ([]net.IP, error)

func ValidatePublicURL(raw string, resolve Resolver) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS are allowed")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("URL hostname is required")
	}
	ips, err := resolve(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("resolve hostname: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("hostname has no addresses")
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return fmt.Errorf("target resolves to a non-public address")
		}
	}
	return nil
}
func blockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	address = address.Unmap()
	for _, prefix := range reservedPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return !address.IsGlobalUnicast()
}

var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}
