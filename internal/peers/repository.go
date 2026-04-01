package peers

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository manages peer persistence in SQLite.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scanPeer(row interface{ Scan(...any) error }) (*Peer, error) {
	p := &Peer{}
	return p, row.Scan(&p.ID, &p.Endpoint, &p.PublicKey, &p.Owner, &p.Status, &p.PinnedAt, &p.LastSeen, &p.AddedAt)
}

func collectPeers(rows *sql.Rows) ([]Peer, error) {
	defer rows.Close()
	var peers []Peer
	for rows.Next() {
		p, err := scanPeer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan peer row: %w", err)
		}
		peers = append(peers, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate peers: %w", err)
	}
	return peers, nil
}

// FindByEndpoint returns the peer with the given endpoint, or nil if not found.
func (r *Repository) FindByEndpoint(ctx context.Context, endpoint string) (*Peer, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, endpoint, public_key, owner, status, pinned_at, last_seen, added_at FROM peers WHERE endpoint = ?`,
		endpoint,
	)
	p, err := scanPeer(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find peer: %w", err)
	}
	return p, nil
}

// SetTrusted upserts a peer as trusted, pinning its public key.
// If the peer already exists as discovered, it is upgraded to trusted.
func (r *Repository) SetTrusted(ctx context.Context, endpoint, publicKey, owner string) (*Peer, error) {
	now := time.Now().Unix()
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO peers (endpoint, public_key, owner, status, pinned_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
			public_key = CASE WHEN peers.status = ? THEN peers.public_key ELSE excluded.public_key END,
			owner      = excluded.owner,
			status     = ?,
			pinned_at  = COALESCE(peers.pinned_at, excluded.pinned_at)
		RETURNING id, endpoint, public_key, owner, status, pinned_at, last_seen, added_at
	`, endpoint, publicKey, owner, string(StatusTrusted), now,
		string(StatusTrusted),
		string(StatusTrusted),
	)
	p, err := scanPeer(row)
	if err != nil {
		return nil, fmt.Errorf("set trusted peer: %w", err)
	}
	return p, nil
}

// AddDiscovered inserts a peer as discovered if not already known. Never downgrades trusted peers.
func (r *Repository) AddDiscovered(ctx context.Context, endpoint, publicKey string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO peers (endpoint, public_key, status)
		VALUES (?, ?, ?)
		ON CONFLICT(endpoint) DO NOTHING
	`, endpoint, publicKey, string(StatusDiscovered))
	if err != nil {
		return fmt.Errorf("add discovered peer: %w", err)
	}
	return nil
}

// All returns every known peer (trusted + discovered).
func (r *Repository) All(ctx context.Context) ([]Peer, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, endpoint, public_key, owner, status, pinned_at, last_seen, added_at FROM peers ORDER BY added_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	return collectPeers(rows)
}

// Trusted returns only peers with status=trusted (exposed via /peer/known for PEX).
func (r *Repository) Trusted(ctx context.Context) ([]Peer, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, endpoint, public_key, owner, status, pinned_at, last_seen, added_at FROM peers WHERE status = ? ORDER BY added_at DESC`,
		string(StatusTrusted),
	)
	if err != nil {
		return nil, fmt.Errorf("list trusted peers: %w", err)
	}
	return collectPeers(rows)
}

// Delete removes a peer by ID.
func (r *Repository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM peers WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete peer: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check rows affected: %w", err)
	}
	return rows > 0, nil
}
