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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage/core"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/crypto/storage"
)

func init() {
	// Initialize SAGE crypto format handlers (JWK, PEM)
	// This allows sage-adk to use all crypto formats supported by SAGE
	sagecrypto.SetFormatConstructors(
		func() sagecrypto.KeyExporter { return formats.NewJWKExporter() },
		func() sagecrypto.KeyExporter { return formats.NewPEMExporter() },
		func() sagecrypto.KeyImporter { return formats.NewJWKImporter() },
		func() sagecrypto.KeyImporter { return formats.NewPEMImporter() },
	)

	// Initialize key generators
	sagecrypto.SetKeyGenerators(
		func() (sagecrypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
		func() (sagecrypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
	)

	// Initialize storage constructors
	sagecrypto.SetStorageConstructors(
		func() sagecrypto.KeyStorage { return storage.NewMemoryKeyStorage() },
	)
}

// Adapter implements the ProtocolAdapter interface for SAGE protocol.
type Adapter struct {
	core            *core.Core
	config          *config.SAGEConfig
	agentDID        string
	signingManager  *SigningManager
	nonceCache      *NonceCache
	didResolver     *DIDResolver
	keyManager      *KeyManager
	privateKey      ed25519.PrivateKey
	networkClient   *NetworkClient
	remoteEndpoint  string // Remote agent endpoint for message transmission
	mu              sync.RWMutex
}

// NewAdapter creates a new SAGE protocol adapter.
func NewAdapter(cfg *config.SAGEConfig) (*Adapter, error) {
	if cfg == nil {
		return nil, errors.ErrConfigurationError.WithMessage("SAGE config is nil")
	}

	if cfg.DID == "" {
		return nil, errors.ErrConfigurationError.WithDetail("field", "DID")
	}

	if cfg.Network == "" {
		return nil, errors.ErrConfigurationError.WithDetail("field", "Network")
	}

	// Initialize SAGE core
	sageCore := core.New()

	// Initialize signing manager for RFC 9421
	signingManager := NewSigningManager()

	// Initialize nonce cache for replay protection (10000 max nonces)
	nonceCache := NewNonceCache(10000)

	// Initialize DID resolver (optional - allows working without blockchain)
	var didResolver *DIDResolver
	// Note: DID resolver initialization requires blockchain connectivity
	// Skip for now - will be initialized when needed

	// Initialize key manager
	keyManager := NewKeyManager()

	// Load private key if path is provided
	var privateKey ed25519.PrivateKey
	if cfg.PrivateKeyPath != "" {
		keyPair, err := keyManager.LoadFromFile(cfg.PrivateKeyPath)
		if err != nil {
			// Private key is optional - log warning but continue
			// This allows the adapter to work without signing capability
		} else {
			// Extract Ed25519 private key
			privateKey, err = keyManager.ExtractEd25519PrivateKey(keyPair)
			if err != nil {
				// Key type mismatch - log warning but continue
			}
		}
	}

	// Initialize network client
	networkClient := NewNetworkClient(nil)

	return &Adapter{
		core:           sageCore,
		config:         cfg,
		agentDID:       cfg.DID,
		signingManager: signingManager,
		nonceCache:     nonceCache,
		didResolver:    didResolver,
		keyManager:     keyManager,
		privateKey:     privateKey,
		networkClient:  networkClient,
	}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "sage"
}

// SetRemoteEndpoint sets the remote agent endpoint for message transmission.
func (a *Adapter) SetRemoteEndpoint(endpoint string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.remoteEndpoint = endpoint
}

// GetRemoteEndpoint returns the configured remote endpoint.
func (a *Adapter) GetRemoteEndpoint() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.remoteEndpoint
}

// SendMessage sends a message using the SAGE protocol.
// Adds security metadata, signs the message, and prepares it for transmission.
func (a *Adapter) SendMessage(ctx context.Context, msg *types.Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Validate message
	if msg == nil {
		return errors.ErrInvalidInput.WithMessage("message is nil")
	}

	if err := msg.Validate(); err != nil {
		return errors.ErrInvalidInput.
			WithMessage("message validation failed").
			WithDetail("error", err.Error())
	}

	// 2. Add security metadata
	if err := a.addSecurityMetadata(msg); err != nil {
		return err
	}

	// 3. Sign message (optional - only if we have key material)
	if err := a.signMessage(msg); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to sign message").
			WithDetail("error", err.Error())
	}

	// 4. Actual network transmission
	if a.remoteEndpoint == "" {
		// No endpoint configured - message is prepared but not sent
		// This allows testing and preparation without network transmission
		return nil
	}

	// Unlock before network call to avoid blocking
	a.mu.Unlock()
	err := a.networkClient.SendMessage(ctx, a.remoteEndpoint, msg)
	a.mu.Lock()

	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to send message over network").
			WithDetail("endpoint", a.remoteEndpoint).
			WithDetail("error", err.Error())
	}

	return nil
}

// ReceiveMessage receives a message using the SAGE protocol.
// Receives message from transport layer and validates security.
func (a *Adapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 1. Receive message from transport layer
	// TODO: Implement transport layer
	// For now, return not implemented error
	return nil, errors.ErrNotImplemented.
		WithMessage("SAGE transport layer not implemented")

	// When transport is implemented, the flow will be:
	// 1. msg := transport.Receive()
	// 2. if err := a.Verify(ctx, msg); err != nil { return nil, err }
	// 3. return msg, nil
}

// Verify verifies a message according to SAGE protocol.
// Performs complete security validation including signature, nonce, and timestamp verification.
func (a *Adapter) Verify(ctx context.Context, msg *types.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 1. Validate security metadata presence
	if msg.Security == nil {
		return errors.ErrInvalidInput.WithMessage("missing security metadata for SAGE protocol")
	}

	// 2. Validate protocol mode
	if msg.Security.Mode != types.ProtocolModeSAGE {
		return errors.ErrProtocolMismatch.
			WithDetail("expected", "SAGE").
			WithDetail("got", string(msg.Security.Mode))
	}

	// 3. Validate security metadata fields
	if err := msg.Security.Validate(); err != nil {
		return errors.ErrInvalidInput.
			WithMessage("security metadata validation failed").
			WithDetail("error", err.Error())
	}

	// 4. Verify timestamp (prevent messages that are too old or too far in future)
	maxClockSkew := 5 * time.Minute // 5 minutes tolerance
	if err := a.signingManager.ValidateTimestamp(msg.Security.Timestamp, maxClockSkew); err != nil {
		return errors.ErrInvalidValue.
			WithMessage("timestamp validation failed").
			WithDetail("error", err.Error())
	}

	// 5. Verify nonce (prevent replay attacks)
	if err := a.nonceCache.Check(msg.Security.Nonce); err != nil {
		return errors.ErrInvalidValue.
			WithMessage("nonce validation failed - possible replay attack").
			WithDetail("nonce", msg.Security.Nonce).
			WithDetail("error", err.Error())
	}

	// 6. Verify signature if present
	if msg.Security.Signature != nil {
		// Resolve DID to get public key
		if a.didResolver == nil {
			return errors.ErrOperationFailed.
				WithMessage("cannot verify signature: DID resolver not available")
		}

		publicKey, err := a.didResolver.ResolvePublicKey(ctx, msg.Security.AgentDID)
		if err != nil {
			return errors.ErrOperationFailed.
				WithMessage("failed to resolve public key for DID").
				WithDetail("did", msg.Security.AgentDID).
				WithDetail("error", err.Error())
		}

		// Type assert to ed25519.PublicKey
		ed25519PubKey, ok := publicKey.(ed25519.PublicKey)
		if !ok {
			return errors.ErrOperationFailed.
				WithMessage("public key is not Ed25519 type").
				WithDetail("did", msg.Security.AgentDID)
		}

		// Verify signature using legacy method
		// TODO: Update to use RFC 9421 when message format supports it
		signatureEnvelope := &SignatureEnvelope{
			Algorithm: string(msg.Security.Signature.Algorithm),
			KeyID:     msg.Security.Signature.KeyID,
			Value:     base64Encode(msg.Security.Signature.Signature),
		}

		if err := a.signingManager.VerifySignature(msg, signatureEnvelope, ed25519PubKey); err != nil {
			return errors.ErrSignatureInvalid.
				WithMessage("signature verification failed").
				WithDetail("error", err.Error())
		}
	}

	return nil
}

// SupportsStreaming returns false as SAGE streaming is not implemented.
func (a *Adapter) SupportsStreaming() bool {
	return false
}

// Stream sends a message and streams the response through the callback.
// Phase 1: Not implemented.
func (a *Adapter) Stream(ctx context.Context, fn protocol.StreamFunc) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return errors.ErrNotImplemented.WithMessage("SAGE streaming not implemented")
}

// Helper functions

// addSecurityMetadata adds SAGE security metadata to a message
func (a *Adapter) addSecurityMetadata(msg *types.Message) error {
	// Generate nonce for replay protection
	nonce, err := generateSecureNonce()
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to generate nonce").
			WithDetail("error", err.Error())
	}

	// Create security metadata
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  a.agentDID,
		Nonce:     nonce,
		Timestamp: time.Now(),
		Sequence:  0, // TODO: Implement sequence counter
	}

	return nil
}

// signMessage signs a message using the signing manager
func (a *Adapter) signMessage(msg *types.Message) error {
	// Check if security metadata exists
	if msg.Security == nil {
		return errors.ErrInvalidInput.WithMessage("security metadata is missing")
	}

	// Check if we have a private key
	if a.privateKey == nil {
		// No private key available - skip signing
		// This is not an error, signing is optional
		return nil
	}

	// Generate key ID from agent DID
	keyID := a.agentDID + "#key-1"

	// Sign the message using the signing manager
	signatureEnvelope, err := a.signingManager.SignMessage(msg, a.privateKey, keyID)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to sign message").
			WithDetail("error", err.Error())
	}

	// Decode signature from base64 to bytes
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureEnvelope.Value)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to decode signature").
			WithDetail("error", err.Error())
	}

	// Add signature to security metadata
	msg.Security.Signature = &types.SignatureData{
		Algorithm:    types.SignatureAlgorithm(signatureEnvelope.Algorithm),
		KeyID:        signatureEnvelope.KeyID,
		Signature:    signatureBytes,
		SignedFields: []string{"message_id", "role", "parts", "timestamp", "nonce"},
	}

	return nil
}

// generateSecureNonce generates a cryptographically secure nonce
func generateSecureNonce() (string, error) {
	// Use combination of timestamp and random bytes
	timestamp := time.Now().UnixNano()

	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Combine timestamp and random bytes
	combined := append([]byte(fmt.Sprintf("%d", timestamp)), randomBytes...)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(combined), nil
}

// base64Encode encodes bytes to base64 string
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
