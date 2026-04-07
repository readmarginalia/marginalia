package common

import (
	"context"
	"fmt"
	"net"
)

// ResolveAndCheck resolves host to IP addresses and rejects loopback,
// private, and link-local targets. Used by both service-level endpoint
// validation and the peerclient transport dial to prevent SSRF.
func ResolveAndCheck(ctx context.Context, host string) ([]net.IP, error) {
	if host == "localhost" {
		return nil, fmt.Errorf("cannot subscribe to localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, checkIP(ip)
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve endpoint: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses for %s", host)
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if err := checkIP(a.IP); err != nil {
			return nil, err
		}
		ips = append(ips, a.IP)
	}
	return ips, nil
}

func checkIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return fmt.Errorf("cannot subscribe to private or reserved address")
	}
	return nil
}
