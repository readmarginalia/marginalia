package http

import (
	"net/netip"
	"strings"
)

var DefaultRealIPHeaders = []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Forwarded-For"}

func RemoteHost(remoteAddr string) netip.Addr {
	host := strings.TrimSpace(remoteAddr)
	if ap, err := netip.ParseAddrPort(host); err == nil {
		return ap.Addr().Unmap()
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap()
	}
	return netip.Addr{}
}

func ForwardedClientIP(header string, value string, trustedRanges []netip.Prefix) netip.Addr {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return netip.Addr{}
	}
	if strings.EqualFold(header, "X-Forwarded-For") {
		parts := strings.Split(candidate, ",")
		// Walk right-to-left: skip entries matching trusted proxy ranges,
		// return the rightmost untrusted entry (the real client IP).
		for i := len(parts) - 1; i >= 0; i-- {
			entry := RemoteHost(strings.TrimSpace(parts[i]))
			if !entry.IsValid() {
				continue
			}
			if IsTrustedIP(entry, trustedRanges) {
				continue
			}
			return entry
		}
		return netip.Addr{}
	}
	return RemoteHost(candidate)
}

func IsTrustedIP(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
