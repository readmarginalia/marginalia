package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Identity holds the node's Ed25519 keypair.
type Identity struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Generate creates a new random Ed25519 keypair.
func Generate() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Identity{PublicKey: pub, PrivateKey: priv}, nil
}

// Encode encodes a key as a base64url string (no padding).
func Encode(key []byte) string {
	return base64.RawURLEncoding.EncodeToString(key)
}

// Decode decodes a base64url string back to raw bytes.
func Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// EncodedPublicKey returns the public key as a base64url string.
func (id *Identity) EncodedPublicKey() string {
	return Encode(id.PublicKey)
}

// Sign signs data with the node's private key.
func (id *Identity) Sign(data []byte) []byte {
	return ed25519.Sign(id.PrivateKey, data)
}

// ParsePublicKey decodes a base64url string and validates it is a well-formed
// Ed25519 public key. Returns an error if the string is malformed or the wrong length.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := Decode(s)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length %d", len(b))
	}
	return ed25519.PublicKey(b), nil
}

// Verify checks a signature against a raw public key and data.
func Verify(pubKey, data, sig []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(pubKey), data, sig)
}
