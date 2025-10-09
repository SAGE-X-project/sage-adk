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
	"sync"

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
// Phase 1: Basic structure with verification only.
type Adapter struct {
	core     *core.Core
	config   *config.SAGEConfig
	agentDID string
	mu       sync.RWMutex
}

// NewAdapter creates a new SAGE protocol adapter.
// Phase 1: Simplified initialization without full DID/crypto setup.
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

	return &Adapter{
		core:     sageCore,
		config:   cfg,
		agentDID: cfg.DID,
	}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "sage"
}

// SendMessage sends a message using the SAGE protocol.
// Phase 1: Not implemented (requires transport layer).
func (a *Adapter) SendMessage(ctx context.Context, msg *types.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return errors.ErrNotImplemented.WithMessage("SAGE transport layer not implemented")
}

// ReceiveMessage receives a message using the SAGE protocol.
// Phase 1: Not implemented (requires transport layer).
func (a *Adapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return nil, errors.ErrNotImplemented.WithMessage("SAGE transport layer not implemented")
}

// Verify verifies a message according to SAGE protocol.
// Checks signature using DID resolution and SAGE core.
func (a *Adapter) Verify(ctx context.Context, msg *types.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if msg.Security == nil {
		return errors.ErrInvalidInput.WithMessage("missing security metadata for SAGE protocol")
	}

	if msg.Security.Mode != types.ProtocolModeSAGE {
		return errors.ErrProtocolMismatch.WithDetail("expected", "SAGE").WithDetail("got", string(msg.Security.Mode))
	}

	if msg.Security.AgentDID == "" {
		return errors.ErrInvalidInput.WithMessage("missing AgentDID in security metadata")
	}

	// Phase 1: Basic validation only
	// Full verification requires:
	// 1. Message serialization
	// 2. Signature verification using SAGE core
	// 3. DID resolution
	// These will be implemented in Phase 2

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
