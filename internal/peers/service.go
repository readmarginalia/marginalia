package peers

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"marginalia/internal/common"
	"marginalia/internal/identity"
	"marginalia/internal/telemetry/logging"
	"net"
	"net/http"
	"strings"
	"time"
)

// PeerInfo is the JSON payload returned by GET /peer/info on a remote peer.
type PeerInfo struct {
	PublicKey string `json:"public_key"`
	Owner     string `json:"owner"`
	RSSUrl    string `json:"rss_url"`
	Version   string `json:"version"`
}

// KnownPeer is a single entry in the PEX exchange list returned by GET /peer/known.
type KnownPeer struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"public_key"`
}

const (
	maxPeerInfoSize = 1 << 20  // 1 MB
	maxPeerKnownSize = 5 << 20 // 5 MB
	maxPEXPeers     = 100
)

type Service struct {
	repo     *Repository
	identity *identity.Identity
	client   *http.Client
}

func NewService(repo *Repository, id *identity.Identity) *Service {
	return &Service{
		repo:     repo,
		identity: id,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: secureDialContext,
			},
		},
	}
}

// secureDialContext closes the DNS rebinding TOCTOU gap
func secureDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := resolveAndCheck(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, ip := range ips {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// Subscribe performs the TOFU handshake + PEX with a remote peer.
// This is the "Add Feed" action: the user provides an endpoint string.
func (s *Service) Subscribe(ctx context.Context, endpoint string) (*Peer, error) {
	logger := logging.FromContext(ctx)
	endpoint = normalizeEndpoint(endpoint)

	if err := validateEndpoint(ctx, endpoint); err != nil {
		return nil, common.ServiceError{Reason: err.Error(), Code: 400}
	}

	// fetch the peer's identity
	info, err := s.fetchPeerInfo(ctx, endpoint)
	if err != nil {
		logger.ErrorContext(ctx, "handshake failed", "endpoint", endpoint, "error", err)
		return nil, common.ServiceError{Reason: "handshake failed", Code: 502}
	}

	// TOFU pin — SQL atomically preserves trusted keys via CASE WHEN
	peer, err := s.repo.SetTrusted(ctx, endpoint, info.PublicKey, info.Owner)
	if err != nil {
		logger.ErrorContext(ctx, "failed to save peer", "endpoint", endpoint, "error", err)
		return nil, common.ServiceError{Reason: "failed to save peer", Code: 500}
	}

	// verify key matches (detects TOFU mismatch race-free)
	if peer.PublicKey != info.PublicKey {
		logger.WarnContext(ctx, "TOFU key mismatch",
			"endpoint", endpoint,
			"pinned_key", peer.PublicKey,
			"received_key", info.PublicKey)
		return nil, common.ServiceError{
			Reason: "key mismatch: peer key changed since first contact",
			Code:   409,
		}
	}

	logger.InfoContext(ctx, "subscribed to peer",
		"endpoint", endpoint,
		"owner", info.Owner,
		"public_key", info.PublicKey)

	// PEX in background — best-effort, must not block the subscribe response
	go func() {
		bctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		s.exchangePeers(bctx, endpoint)
	}()

	return peer, nil
}

// All returns every known peer.
func (s *Service) All(ctx context.Context) ([]Peer, error) {
	peers, err := s.repo.All(ctx)
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx, "failed to list peers", "error", err)
		return nil, common.ServiceError{Reason: "failed to list peers", Code: 500}
	}
	return peers, nil
}

// Trusted returns only trusted peers (for PEX responses).
func (s *Service) Trusted(ctx context.Context) ([]Peer, error) {
	peers, err := s.repo.Trusted(ctx)
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx, "failed to list trusted peers", "error", err)
		return nil, common.ServiceError{Reason: "failed to list trusted peers", Code: 500}
	}
	return peers, nil
}

// Delete removes a peer by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	found, err := s.repo.Delete(ctx, id)
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx, "failed to delete peer", "id", id, "error", err)
		return common.ServiceError{Reason: "failed to delete peer", Code: 500}
	}
	if !found {
		return common.ServiceError{Reason: "peer not found", Code: 404}
	}
	return nil
}

func (s *Service) fetchPeerInfo(ctx context.Context, endpoint string) (*PeerInfo, error) {
	url := endpoint + "/peer/info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned %d", resp.StatusCode)
	}

	var info PeerInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPeerInfoSize)).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if info.PublicKey == "" {
		return nil, fmt.Errorf("peer returned empty public key")
	}

	keyBytes, err := identity.Decode(info.PublicKey)
	if err != nil || len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("peer returned invalid public key")
	}

	return &info, nil
}

func (s *Service) exchangePeers(ctx context.Context, endpoint string) {
	logger := logging.FromContext(ctx)

	url := endpoint + "/peer/known"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.DebugContext(ctx, "PEX request build failed", "endpoint", endpoint, "error", err)
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		logger.DebugContext(ctx, "PEX fetch failed", "endpoint", endpoint, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.DebugContext(ctx, "PEX returned non-OK status", "endpoint", endpoint, "status", resp.StatusCode)
		return
	}

	var known []KnownPeer
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPeerKnownSize)).Decode(&known); err != nil {
		logger.DebugContext(ctx, "PEX decode failed", "endpoint", endpoint, "error", err)
		return
	}

	if len(known) > maxPEXPeers {
		known = known[:maxPEXPeers]
	}

	added := 0
	for _, kp := range known {
		if kp.Endpoint == "" || kp.PublicKey == "" {
			logger.DebugContext(ctx, "PEX skipped empty entry", "from", endpoint)
			continue
		}
		// Validate key format
		keyBytes, err := identity.Decode(kp.PublicKey)
		if err != nil || len(keyBytes) != ed25519.PublicKeySize {
			logger.DebugContext(ctx, "PEX skipped invalid key", "from", endpoint, "key", kp.PublicKey)
			continue
		}
		// Validate endpoint against SSRF
		normalized := normalizeEndpoint(kp.Endpoint)
		if err := validateEndpoint(ctx, normalized); err != nil {
			logger.DebugContext(ctx, "PEX skipped invalid endpoint", "from", endpoint, "peer_endpoint", kp.Endpoint, "reason", err)
			continue
		}
		if err := s.repo.AddDiscovered(ctx, normalized, kp.PublicKey); err != nil {
			logger.ErrorContext(ctx, "PEX failed to persist peer", "endpoint", normalized, "error", err)
			continue
		}
		added++
	}

	if added > 0 {
		logger.InfoContext(ctx, "PEX discovered new peers", "from", endpoint, "count", added)
	}
}

// normalizeEndpoint ensures the endpoint has a port, defaulting to 9595.
func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimRight(endpoint, "/")

	var scheme, hostport string
	switch {
	case strings.HasPrefix(endpoint, "https://"):
		scheme, hostport = "https", strings.TrimPrefix(endpoint, "https://")
	case strings.HasPrefix(endpoint, "http://"):
		scheme, hostport = "http", strings.TrimPrefix(endpoint, "http://")
	default:
		scheme, hostport = "http", endpoint
	}

	if _, _, err := net.SplitHostPort(hostport); err != nil {
		hostport += ":9595"
	}
	return scheme + "://" + hostport
}

func resolveAndCheck(ctx context.Context, host string) ([]net.IP, error) {
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

// validateEndpoint rejects loopback, private, and link-local addresses.
func validateEndpoint(ctx context.Context, endpoint string) error {
	hostport := endpoint
	if _, after, ok := strings.Cut(endpoint, "://"); ok {
		hostport = after
	}
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}
	_, err = resolveAndCheck(ctx, host)
	return err
}

func checkIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return fmt.Errorf("cannot subscribe to private or reserved address")
	}
	return nil
}
