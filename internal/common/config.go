package common

import (
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

func EnvBool(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		slog.Error("invalid boolean value",
			"name", name,
			"value", value,
			"error", err)

		os.Exit(1)
	}
	return enabled
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

func MustParseTrustedProxyRanges(values []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		if addr, err := netip.ParseAddr(value); err == nil {
			addr = addr.Unmap()
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			slog.Error("invalid TRUSTED_PROXIES entry",
				"value", value,
				"error", err)
			os.Exit(1)
		}
		prefix = prefix.Masked()
		if prefix.Addr().Is4In6() {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}
