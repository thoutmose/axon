package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Sealer encrypts small secrets at rest with AES-256-GCM.
type Sealer struct{ aead cipher.AEAD }

// NewSealer takes a 64-character hex key (32 bytes), e.g. `openssl rand -hex 32`.
func NewSealer(hexKey string) (*Sealer, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("AXON_ENCRYPTION_KEY must be 64 hex characters (32 bytes)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead}, nil
}

// Seal returns nonce || ciphertext.
func (s *Sealer) Seal(plaintext []byte) []byte {
	nonce := make([]byte, s.aead.NonceSize())
	_, _ = rand.Read(nonce) // crypto/rand.Read never fails on supported platforms
	return s.aead.Seal(nonce, nonce, plaintext, nil)
}

// Open reverses Seal.
func (s *Sealer) Open(sealed []byte) ([]byte, error) {
	n := s.aead.NonceSize()
	if len(sealed) < n {
		return nil, fmt.Errorf("sealed value too short")
	}
	return s.aead.Open(nil, sealed[:n], sealed[n:], nil)
}
