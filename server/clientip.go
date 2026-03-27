package server

import (
	"log"
	"net/http"
	"net/netip"
	"strings"
)

var defaultRealIPHeaders = []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Forwarded-For"}

// AuthConfig holds authentication and client identity settings.
type AuthConfig struct {
	Token              string
	EnableRateLimit    bool
	TrustProxy         bool
	RealIPHeaders      []string
	TrustedProxyRanges []netip.Prefix
}

func (cfg AuthConfig) withDefaults() AuthConfig {
	if len(cfg.RealIPHeaders) == 0 {
		cfg.RealIPHeaders = append([]string(nil), defaultRealIPHeaders...)
	}
	return cfg
}

func (cfg AuthConfig) clientIdentity(r *http.Request) (string, bool) {
	peer := remoteHost(r.RemoteAddr)
	if cfg.usesTrustedProxy(peer) {
		for _, header := range cfg.RealIPHeaders {
			if clientIP := forwardedClientIP(header, r.Header.Get(header), cfg.TrustedProxyRanges); clientIP.IsValid() {
				return clientIP.String(), true
			}
		}
		log.Printf("proxy warning: peer %s is trusted but no valid client IP found in headers %v, falling back to peer address", r.RemoteAddr, cfg.RealIPHeaders)
	}
	if peer.IsValid() {
		return peer.String(), false
	}
	return strings.TrimSpace(r.RemoteAddr), false
}

func (cfg AuthConfig) usesTrustedProxy(peer netip.Addr) bool {
	if !cfg.TrustProxy {
		return false
	}
	if len(cfg.TrustedProxyRanges) == 0 {
		return true
	}
	return isTrustedIP(peer, cfg.TrustedProxyRanges)
}

func remoteHost(remoteAddr string) netip.Addr {
	host := strings.TrimSpace(remoteAddr)
	if ap, err := netip.ParseAddrPort(host); err == nil {
		return ap.Addr().Unmap()
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap()
	}
	return netip.Addr{}
}

func forwardedClientIP(header string, value string, trustedRanges []netip.Prefix) netip.Addr {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return netip.Addr{}
	}
	if strings.EqualFold(header, "X-Forwarded-For") {
		parts := strings.Split(candidate, ",")
		// Walk right-to-left: skip entries matching trusted proxy ranges,
		// return the rightmost untrusted entry (the real client IP).
		for i := len(parts) - 1; i >= 0; i-- {
			entry := remoteHost(strings.TrimSpace(parts[i]))
			if !entry.IsValid() {
				continue
			}
			if isTrustedIP(entry, trustedRanges) {
				continue
			}
			return entry
		}
		return netip.Addr{}
	}
	return remoteHost(candidate)
}


func isTrustedIP(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
