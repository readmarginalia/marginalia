package http

import (
	"marginalia/internal/configuration"
	"net/netip"
	"strings"

	stdhttp "net/http"
)

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

func ClientIdentity(r *stdhttp.Request, cfg configuration.AppConfig) (identity string, proxied bool) {
	peer := RemoteHost(r.RemoteAddr)
	if usesTrustedProxy(peer, cfg) {
		for _, header := range cfg.RealIPHeaders {
			if clientIP := ForwardedClientIP(header, r.Header.Get(header), cfg.TrustedProxyRanges); clientIP.IsValid() {
				return clientIP.String(), true
			}
		}
	}
	if peer.IsValid() {
		return peer.String(), false
	}
	return strings.TrimSpace(r.RemoteAddr), false
}

func usesTrustedProxy(peer netip.Addr, cfg configuration.AppConfig) bool {
	if !cfg.TrustProxy {
		return false
	}
	if len(cfg.TrustedProxyRanges) == 0 {
		return true
	}
	return IsTrustedIP(peer, cfg.TrustedProxyRanges)
}
