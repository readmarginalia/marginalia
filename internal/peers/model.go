package peers

// Status represents the trust level of a known peer.
type Status string

const (
	// StatusTrusted means the user explicitly subscribed to this peer; their key is pinned.
	StatusTrusted Status = "trusted"
	// StatusDiscovered means this peer was received via PEX; not yet explicitly trusted.
	StatusDiscovered Status = "discovered"
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

// ToKnownList projects a slice of Peers into the minimal KnownPeer form
// advertised via PEX. Keeping the projection here ensures only the intended
// fields are exposed, regardless of how many callers convert the list.
func ToKnownList(ps []Peer) []KnownPeer {
	known := make([]KnownPeer, len(ps))
	for i, p := range ps {
		known[i] = KnownPeer{Endpoint: p.Endpoint, PublicKey: p.PublicKey}
	}
	return known
}

// Peer is a remote marginalia node the local instance knows about.
type Peer struct {
	ID        int64   `json:"id"`
	Endpoint  string  `json:"endpoint"`        // "192.168.1.50:9595" or "rss.friend.com"
	PublicKey string  `json:"public_key"`      // base64url Ed25519 public key
	Owner     *string `json:"owner,omitempty"` // display name from /peer/info
	Status    Status  `json:"status"`
	PinnedAt  *int64  `json:"pinned_at,omitempty"` // unix timestamp, set when status=trusted
	LastSeen  *int64  `json:"last_seen,omitempty"` // unix timestamp of last successful contact
	AddedAt   int64   `json:"added_at"`
}
