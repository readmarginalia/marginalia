package server

import (
	"log"
	"net"
	"net/http"
	"strings"
)

var defaultRealIPHeaders = []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Forwarded-For"}

// AuthConfig holds authentication and client identity settings.
type AuthConfig struct {
	Token              string
	EnableRateLimit    bool
	TrustProxy         bool
	RealIPHeaders      []string
	TrustedProxyRanges []*net.IPNet
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
			if clientIP := forwardedClientIP(header, r.Header.Get(header), cfg.TrustedProxyRanges); clientIP != "" {
				return clientIP, true
			}
		}
		log.Printf("proxy warning: peer=%s is trusted but no valid client IP found in headers %v, falling back to peer address", peer, cfg.RealIPHeaders)
	}
	if peer != "" {
		return peer, false
	}
	return strings.TrimSpace(r.RemoteAddr), false
}

func (cfg AuthConfig) usesTrustedProxy(peer string) bool {
	if !cfg.TrustProxy {
		return false
	}
	if len(cfg.TrustedProxyRanges) == 0 {
		return true
	}
	return isTrustedIP(peer, cfg.TrustedProxyRanges)
}

func remoteHost(remoteAddr string) string {
	host := strings.TrimSpace(remoteAddr)
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

func forwardedClientIP(header string, value string, trustedRanges []*net.IPNet) string {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return ""
	}
	if strings.EqualFold(header, "X-Forwarded-For") {
		parts := strings.Split(candidate, ",")
		// Walk right-to-left: skip entries matching trusted proxy ranges,
		// return the rightmost untrusted entry (the real client IP).
		for i := len(parts) - 1; i >= 0; i-- {
			entry := parseIPFromForwardedEntry(strings.TrimSpace(parts[i]))
			if entry == "" {
				continue
			}
			if isTrustedIP(entry, trustedRanges) {
				continue
			}
			return entry
		}
		return ""
	}
	return parseIPFromForwardedEntry(candidate)
}

func parseIPFromForwardedEntry(entry string) string {
	if host, _, err := net.SplitHostPort(entry); err == nil {
		entry = host
	}
	entry = strings.TrimPrefix(strings.TrimSuffix(entry, "]"), "[")
	if ip := net.ParseIP(entry); ip != nil {
		return ip.String()
	}
	return ""
}

func isTrustedIP(ipStr string, trustedRanges []*net.IPNet) bool {
	if len(trustedRanges) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, cidr := range trustedRanges {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
