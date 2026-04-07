package peerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"marginalia/internal/common"
	"marginalia/internal/identity"
	"marginalia/internal/peers"
	"marginalia/internal/telemetry/logging"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	maxPeerInfoSize  = 1 << 20 // 1 MB
	maxPeerKnownSize = 5 << 20 // 5 MB

	userAgent = "Marginalia/1.0"
)

var dialer = &net.Dialer{}

// Client is an HTTP client for calling remote Marginalia peer nodes.
// A single Client should be shared across all peer calls — its underlying
// transport maintains a per-host connection pool for connection reuse.
type Client struct {
	client *http.Client
}

// New returns a Client whose transport enforces SSRF protection via
// secureDialContext — private, loopback, and link-local addresses are
// rejected at the TCP dial layer as defense-in-depth.
func New(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
			Transport: otelhttp.NewTransport(&http.Transport{
				DialContext: secureDialContext,
			}),
		},
	}
}

// FetchInfo calls GET /peer/info on the given endpoint and returns the
// parsed PeerInfo. The response public key is validated as a well-formed
// Ed25519 key before returning.
func (c *Client) FetchInfo(ctx context.Context, endpoint string) (*peers.PeerInfo, error) {
	resp, err := c.get(ctx, endpoint, "peer/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		logging.FromContext(ctx).ErrorContext(ctx, "peerclient: peer/info returned error status", "status", resp.StatusCode, "endpoint", endpoint)
		return nil, fmt.Errorf("peer returned %d", resp.StatusCode)
	}

	var info peers.PeerInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPeerInfoSize)).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if info.PublicKey == "" {
		return nil, fmt.Errorf("peer returned empty public key")
	}
	if _, err := identity.ParsePublicKey(info.PublicKey); err != nil {
		return nil, fmt.Errorf("peer returned invalid public key")
	}

	return &info, nil
}

// FetchKnown calls GET /peer/known on the given endpoint and returns the
// raw list of known peers. Callers are responsible for validating and
// persisting the returned entries.
func (c *Client) FetchKnown(ctx context.Context, endpoint string) ([]peers.KnownPeer, error) {
	resp, err := c.get(ctx, endpoint, "peer/known")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		logging.FromContext(ctx).DebugContext(ctx, "peerclient: peer/known returned non-OK status", "status", resp.StatusCode, "endpoint", endpoint)
		return nil, fmt.Errorf("peer returned %d", resp.StatusCode)
	}

	var known []peers.KnownPeer
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPeerKnownSize)).Decode(&known); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return known, nil
}

// get builds and executes a GET request to the given path on endpoint.
// It handles URL construction, User-Agent, and connect-level errors.
// Callers must close the response body and check the status code.
func (c *Client) get(ctx context.Context, endpoint, relPath string) (*http.Response, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	u.Path = path.Join(u.Path, relPath)

	logger := logging.FromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		logger.ErrorContext(ctx, "peerclient: request build failed", "endpoint", endpoint, "error", err)
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "peerclient: connect failed", "endpoint", endpoint, "error", err)
		return nil, fmt.Errorf("connect: %w", err)
	}
	return resp, nil
}

// secureDialContext is a custom dialer that resolves the target hostname and
// rejects connections to private, loopback, and link-local addresses before
// establishing the TCP connection. This closes the DNS rebinding TOCTOU gap
// as defense-in-depth alongside service-level endpoint validation.
func secureDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := common.ResolveAndCheck(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
