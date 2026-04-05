package peers

import (
	"context"
	"fmt"
	"marginalia/internal/common"
	"marginalia/internal/identity"
	"marginalia/internal/telemetry/logging"
	"net"
	"strings"
	"time"
)

const maxPEXPeers = 100

// HTTPClient is the interface for fetching data from remote Marginalia peer nodes.
// The concrete implementation lives in internal/interop/peerclient.
type HTTPClient interface {
	FetchInfo(ctx context.Context, endpoint string) (*PeerInfo, error)
	FetchKnown(ctx context.Context, endpoint string) ([]KnownPeer, error)
}

type Service struct {
	repo       *Repository
	httpClient HTTPClient
}

func NewService(repo *Repository, httpClient HTTPClient) *Service {
	return &Service{
		repo:       repo,
		httpClient: httpClient,
	}
}

// Subscribe performs the TOFU handshake + PEX with a remote peer.
// This is the "Add Feed" action: the user provides an endpoint string.
func (s *Service) Subscribe(ctx context.Context, endpoint string) (*Peer, error) {
	logger := logging.FromContext(ctx)
	endpoint = normalizeEndpoint(endpoint)

	if err := validateEndpoint(ctx, endpoint); err != nil {
		return nil, common.ServiceError{Reason: err.Error(), Code: 400}
	}

	info, err := s.httpClient.FetchInfo(ctx, endpoint)
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
		bctx = logging.WithLogger(bctx, logger)
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

func (s *Service) exchangePeers(ctx context.Context, endpoint string) {
	logger := logging.FromContext(ctx)

	known, err := s.httpClient.FetchKnown(ctx, endpoint)
	if err != nil {
		logger.WarnContext(ctx, "PEX fetch failed, peer graph will not grow", "endpoint", endpoint, "error", err)
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
		if _, err := identity.ParsePublicKey(kp.PublicKey); err != nil {
			logger.DebugContext(ctx, "PEX skipped invalid key", "from", endpoint, "key", kp.PublicKey)
			continue
		}
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
	} else if len(known) > 0 {
		logger.WarnContext(ctx, "PEX received peers but none were persisted", "from", endpoint, "received", len(known))
	}
}

// normalizeEndpoint ensures the endpoint has a scheme and port, defaulting to 9595.
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
		// Bare IPv6 address (e.g. "2001:db8::1") has no brackets and no port;
		// wrap it so SplitHostPort can parse it after we append the default port.
		if strings.Contains(hostport, ":") && !strings.HasPrefix(hostport, "[") {
			hostport = "[" + hostport + "]"
		}
		hostport += ":9595"
	}
	return scheme + "://" + hostport
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
	_, err = common.ResolveAndCheck(ctx, host)
	return err
}
