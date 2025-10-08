// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package sage

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"time"

	"lukechampine.com/blake3"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// SigningManager handles RFC 9421 message signing and verification.
type SigningManager struct {
	// No state needed - stateless operations
}

// NewSigningManager creates a new signing manager.
func NewSigningManager() *SigningManager {
	return &SigningManager{}
}

// SignMessage signs a message using Ed25519 and RFC 9421 format.
func (sm *SigningManager) SignMessage(message interface{}, privateKey ed25519.PrivateKey, keyID string) (*SignatureEnvelope, error) {
	if privateKey == nil {
		return nil, errors.ErrInvalidInput.WithMessage("private key is nil")
	}

	if keyID == "" {
		return nil, errors.ErrInvalidInput.WithMessage("key ID is empty")
	}

	// Create signature base string
	signatureBase, err := sm.createSignatureBase(message)
	if err != nil {
		return nil, err
	}

	// Hash with BLAKE3
	hash := blake3.Sum256([]byte(signatureBase))

	// Sign with Ed25519
	signature := ed25519.Sign(privateKey, hash[:])

	return &SignatureEnvelope{
		Algorithm: "EdDSA",
		KeyID:     keyID,
		Value:     base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// VerifySignature verifies a message signature using Ed25519 and RFC 9421.
func (sm *SigningManager) VerifySignature(message interface{}, signature *SignatureEnvelope, publicKey ed25519.PublicKey) error {
	if signature == nil {
		return errors.ErrInvalidInput.WithMessage("signature is nil")
	}

	if publicKey == nil {
		return errors.ErrInvalidInput.WithMessage("public key is nil")
	}

	// Decode signature from base64
	signatureBytes, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return errors.ErrInvalidInput.
			WithMessage("failed to decode signature").
			WithDetail("error", err.Error())
	}

	// Create signature base string
	signatureBase, err := sm.createSignatureBase(message)
	if err != nil {
		return err
	}

	// Hash with BLAKE3
	hash := blake3.Sum256([]byte(signatureBase))

	// Verify signature
	if !ed25519.Verify(publicKey, hash[:], signatureBytes) {
		return errors.ErrSignatureInvalid.WithMessage("signature verification failed")
	}

	return nil
}

// createSignatureBase creates the RFC 9421 signature base string.
// This canonicalizes the message fields into a standardized format for signing.
func (sm *SigningManager) createSignatureBase(message interface{}) (string, error) {
	// For messages with a Signature field, we need to exclude it from the signature base
	// Otherwise the signature would be signing itself, which is impossible
	messageToSign := message

	// Check if message has a Signature field and create a copy without it
	switch v := message.(type) {
	case *HandshakeRequest:
		// Create a copy without signature
		copy := *v
		copy.Signature = SignatureEnvelope{}
		messageToSign = copy
	case *HandshakeResponse:
		copy := *v
		copy.Signature = SignatureEnvelope{}
		messageToSign = copy
	case *HandshakeComplete:
		copy := *v
		copy.Signature = SignatureEnvelope{}
		messageToSign = copy
	case *ApplicationMessage:
		copy := *v
		copy.Signature = SignatureEnvelope{}
		messageToSign = copy
	case HandshakeRequest:
		v.Signature = SignatureEnvelope{}
		messageToSign = v
	case HandshakeResponse:
		v.Signature = SignatureEnvelope{}
		messageToSign = v
	case HandshakeComplete:
		v.Signature = SignatureEnvelope{}
		messageToSign = v
	case ApplicationMessage:
		v.Signature = SignatureEnvelope{}
		messageToSign = v
	}

	// Serialize message to JSON with deterministic ordering
	messageJSON, err := json.Marshal(messageToSign)
	if err != nil {
		return "", errors.ErrOperationFailed.
			WithMessage("failed to marshal message").
			WithDetail("error", err.Error())
	}

	// Use the entire JSON as the signature base for simplicity and security
	// This ensures any change to the message invalidates the signature
	hash := blake3.Sum256(messageJSON)
	signatureBase := base64.StdEncoding.EncodeToString(hash[:])

	return signatureBase, nil
}

// ValidateTimestamp validates that a timestamp is within acceptable clock skew.
func (sm *SigningManager) ValidateTimestamp(timestamp time.Time, maxClockSkew time.Duration) error {
	now := time.Now()
	diff := now.Sub(timestamp)
	if diff < 0 {
		diff = -diff
	}

	if diff > maxClockSkew {
		return errors.ErrOperationFailed.
			WithMessage("timestamp outside acceptable clock skew").
			WithDetail("diff", diff.String()).
			WithDetail("max_skew", maxClockSkew.String())
	}

	return nil
}

// NonceCache provides replay protection by tracking used nonces.
type NonceCache struct {
	nonces map[string]time.Time
	maxSize int
}

// NewNonceCache creates a new nonce cache.
func NewNonceCache(maxSize int) *NonceCache {
	return &NonceCache{
		nonces:  make(map[string]time.Time),
		maxSize: maxSize,
	}
}

// Check checks if a nonce has been used before.
// Returns error if nonce is replayed.
func (nc *NonceCache) Check(nonce string) error {
	if _, exists := nc.nonces[nonce]; exists {
		return errors.ErrOperationFailed.
			WithMessage("nonce replay detected").
			WithDetail("nonce", nonce)
	}

	// Add nonce
	nc.nonces[nonce] = time.Now()

	// Cleanup if cache is too large
	if len(nc.nonces) > nc.maxSize {
		nc.cleanup()
	}

	return nil
}

// cleanup removes oldest nonces when cache is full.
func (nc *NonceCache) cleanup() {
	// Find oldest timestamp
	var oldestTime time.Time
	var oldestNonce string
	first := true

	for nonce, t := range nc.nonces {
		if first || t.Before(oldestTime) {
			oldestTime = t
			oldestNonce = nonce
			first = false
		}
	}

	// Remove oldest
	if oldestNonce != "" {
		delete(nc.nonces, oldestNonce)
	}
}
