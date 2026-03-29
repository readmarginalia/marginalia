package auth

import (
	"log"
	"marginalia/internal/infra/http"
	stdhttp "net/http"
	"net/netip"
	"strings"
)

type AuthConfig struct {
	Token              string
	EnableRateLimit    bool
	TrustProxy         bool
	RealIPHeaders      []string
	TrustedProxyRanges []netip.Prefix
}

func (cfg AuthConfig) WithDefaults() AuthConfig {
	if len(cfg.RealIPHeaders) == 0 {
		cfg.RealIPHeaders = append([]string(nil), http.DefaultRealIPHeaders...)
	}
	return cfg
}

func (cfg AuthConfig) clientIdentity(r *stdhttp.Request) (string, bool) {
	peer := http.RemoteHost(r.RemoteAddr)
	if cfg.usesTrustedProxy(peer) {
		for _, header := range cfg.RealIPHeaders {
			if clientIP := http.ForwardedClientIP(header, r.Header.Get(header), cfg.TrustedProxyRanges); clientIP.IsValid() {
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
	return http.IsTrustedIP(peer, cfg.TrustedProxyRanges)
}
