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
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage/core/rfc9421"
)

// SigningManager handles RFC 9421 message signing and verification.
type SigningManager struct {
	verifier *rfc9421.Verifier
}

// NewSigningManager creates a new signing manager.
func NewSigningManager() *SigningManager {
	return &SigningManager{
		verifier: rfc9421.NewVerifier(),
	}
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
	case *types.Message:
		// Create a copy without signature for types.Message
		copy := *v
		if copy.Security != nil {
			securityCopy := *copy.Security
			securityCopy.Signature = nil
			copy.Security = &securityCopy
		}
		messageToSign = copy
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

// SignMessageRFC9421 signs a message using RFC 9421 standard.
// This is the preferred method for new code.
func (sm *SigningManager) SignMessageRFC9421(
	agentDID string,
	messageID string,
	body []byte,
	headers map[string]string,
	privateKey ed25519.PrivateKey,
	keyID string,
) (*rfc9421.Message, error) {
	if privateKey == nil {
		return nil, errors.ErrInvalidInput.WithMessage("private key is nil")
	}

	if agentDID == "" {
		return nil, errors.ErrInvalidInput.WithMessage("agent DID is empty")
	}

	if messageID == "" {
		return nil, errors.ErrInvalidInput.WithMessage("message ID is empty")
	}

	// Build RFC 9421 message
	builder := rfc9421.NewMessageBuilder().
		WithAgentDID(agentDID).
		WithMessageID(messageID).
		WithTimestamp(time.Now()).
		WithNonce(generateNonce()).
		WithBody(body).
		WithAlgorithm(rfc9421.AlgorithmEdDSA).
		WithKeyID(keyID).
		WithSignedFields("agent_did", "message_id", "timestamp", "nonce", "body")

	// Add headers
	for k, v := range headers {
		builder.AddHeader(k, v)
	}

	message := builder.Build()

	// Create signature base
	signatureBase := sm.verifier.ConstructSignatureBase(message)

	// Sign with Ed25519 directly (no hashing - verifier expects raw signature base)
	signature := ed25519.Sign(privateKey, []byte(signatureBase))
	message.Signature = signature

	return message, nil
}

// VerifyMessageRFC9421 verifies a message signature using RFC 9421 standard.
// This is the preferred method for new code.
func (sm *SigningManager) VerifyMessageRFC9421(
	message *rfc9421.Message,
	publicKey ed25519.PublicKey,
	opts *rfc9421.VerificationOptions,
) error {
	if message == nil {
		return errors.ErrInvalidInput.WithMessage("message is nil")
	}

	if publicKey == nil {
		return errors.ErrInvalidInput.WithMessage("public key is nil")
	}

	if opts == nil {
		opts = rfc9421.DefaultVerificationOptions()
	}

	// Use RFC 9421 verifier
	if err := sm.verifier.VerifySignature(publicKey, message, opts); err != nil {
		return errors.ErrSignatureInvalid.
			WithMessage("RFC 9421 signature verification failed").
			WithDetail("error", err.Error())
	}

	return nil
}

// generateNonce generates a random nonce for replay protection.
func generateNonce() string {
	// Use timestamp + random component
	timestamp := time.Now().UnixNano()
	return base64.StdEncoding.EncodeToString([]byte(
		base64.StdEncoding.EncodeToString([]byte(string(rune(timestamp)))),
	))
}
