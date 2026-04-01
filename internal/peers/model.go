package peers

// Status represents the trust level of a known peer.
type Status string

const (
	// StatusTrusted means the user explicitly subscribed to this peer; their key is pinned.
	StatusTrusted Status = "trusted"
	// StatusDiscovered means this peer was received via PEX; not yet explicitly trusted.
	StatusDiscovered Status = "discovered"
)

// Peer is a remote marginalia node the local instance knows about.
type Peer struct {
	ID        int64   `json:"id"`
	Endpoint  string  `json:"endpoint"`            // "192.168.1.50:9595" or "rss.friend.com"
	PublicKey string  `json:"public_key"`          // base64url Ed25519 public key
	Owner     *string `json:"owner,omitempty"`     // display name from /peer/info
	Status    Status  `json:"status"`
	PinnedAt  *int64  `json:"pinned_at,omitempty"` // unix timestamp, set when status=trusted
	LastSeen  *int64  `json:"last_seen,omitempty"` // unix timestamp of last successful contact
	AddedAt   int64   `json:"added_at"`
}
