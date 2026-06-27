package configuration

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

var DefaultRealIPHeaders = []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Forwarded-For"}

type AppConfig struct {
	Environment        string
	Token              string
	Owner              string
	Port               string
	DbPath             string
	ThemeName          string
	TrustProxy         bool
	RealIPHeaders      []string
	TrustedProxyRanges []netip.Prefix
	OtelEndpoint       string
	AuthRateLimit      bool
}

func Load() (AppConfig, error) {
	token := os.Getenv("TOKEN")
	if token == "" {
		return AppConfig{}, fmt.Errorf("TOKEN is required")
	}

	owner := os.Getenv("OWNER")
	themeName := os.Getenv("THEME_NAME")
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	port := os.Getenv("PORT")
	if port == "" {
		port = "9595"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/marginalia.db"
	}

	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	authRateLimit, err := EnvBool("AUTH_RATE_LIMIT")
	if err != nil {
		return AppConfig{}, err
	}
	trustProxy, err := EnvBool("TRUST_PROXY")
	if err != nil {
		return AppConfig{}, err
	}
	realIPHeaders := EnvList("REAL_IP_HEADERS")
	if len(realIPHeaders) == 0 {
		realIPHeaders = DefaultRealIPHeaders
	}

	trustedProxyRanges, err := ParseTrustedProxyRanges(EnvList("TRUSTED_PROXIES"))
	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		Environment:        environment,
		Token:              token,
		Owner:              owner,
		Port:               port,
		DbPath:             dbPath,
		ThemeName:          themeName,
		OtelEndpoint:       otelEndpoint,
		AuthRateLimit:      authRateLimit,
		TrustProxy:         trustProxy,
		RealIPHeaders:      realIPHeaders,
		TrustedProxyRanges: trustedProxyRanges,
	}, nil
}

func EnvBool(name string) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid boolean value for %s: %s", name, value)
	}
	return enabled, nil
}

func EnvList(name string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func ParseTrustedProxyRanges(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		if addr, err := netip.ParseAddr(value); err == nil {
			addr = addr.Unmap()
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy range: %s", value)
		}
		prefix = prefix.Masked()
		if prefix.Addr().Is4In6() {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}
