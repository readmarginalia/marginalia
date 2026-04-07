package identity

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"
)

// Repository persists the local node identity in SQLite.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Load returns the stored identity, or nil if none exists yet.
func (r *Repository) Load(ctx context.Context) (*Identity, error) {
	var pubEnc, privEnc string
	err := r.db.QueryRowContext(ctx, `SELECT public_key, private_key FROM identity WHERE id = 1`).
		Scan(&pubEnc, &privEnc)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}

	pubBytes, err := Decode(pubEnc)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	privBytes, err := Decode(privEnc)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("stored public key has invalid length %d", len(pubBytes))
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("stored private key has invalid length %d", len(privBytes))
	}

	return &Identity{
		PublicKey:  ed25519.PublicKey(pubBytes),
		PrivateKey: ed25519.PrivateKey(privBytes),
	}, nil
}

// Save persists the identity. Safe to call on every startup; replaces if already present.
func (r *Repository) Save(ctx context.Context, id *Identity) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO identity (id, public_key, private_key) VALUES (1, ?, ?)`,
		Encode(id.PublicKey),
		Encode(id.PrivateKey),
	)
	if err != nil {
		return fmt.Errorf("save identity: %w", err)
	}
	return nil
}
