package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func envBool(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatalf("invalid %s: %v", name, err)
	}
	return enabled
}

func envList(name string) []string {
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

func mustParseTrustedProxyRanges(values []string) []*net.IPNet {
	ranges := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		if ip := net.ParseIP(value); ip != nil {
			maskBits := 32
			if ip.To4() == nil {
				maskBits = 128
			}
			ranges = append(ranges, &net.IPNet{IP: ip, Mask: net.CIDRMask(maskBits, maskBits)})
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			log.Fatalf("invalid TRUSTED_PROXIES entry %q: %v", value, err)
		}
		ranges = append(ranges, network)
	}
	return ranges
}
