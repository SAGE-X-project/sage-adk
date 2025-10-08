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

package types

import (
	"fmt"
	"time"
)

// ProtocolMode represents the protocol mode for message processing.
type ProtocolMode string

const (
	// ProtocolModeA2A indicates A2A-only mode (no security).
	ProtocolModeA2A ProtocolMode = "a2a"
	// ProtocolModeSAGE indicates SAGE security mode.
	ProtocolModeSAGE ProtocolMode = "sage"
	// ProtocolModeAuto indicates automatic protocol detection.
	ProtocolModeAuto ProtocolMode = "auto"
)

// IsValid checks if the protocol mode is valid.
func (p ProtocolMode) IsValid() bool {
	return p == ProtocolModeA2A || p == ProtocolModeSAGE || p == ProtocolModeAuto
}

// SignatureAlgorithm represents supported signature algorithms.
type SignatureAlgorithm string

const (
	// AlgorithmEdDSA is Ed25519 signature algorithm.
	AlgorithmEdDSA SignatureAlgorithm = "EdDSA"
	// AlgorithmES256K is ECDSA with secp256k1 curve.
	AlgorithmES256K SignatureAlgorithm = "ES256K"
	// AlgorithmECDSA is generic ECDSA.
	AlgorithmECDSA SignatureAlgorithm = "ECDSA"
	// AlgorithmECDSASecp256k1 is ECDSA with secp256k1 curve (explicit).
	AlgorithmECDSASecp256k1 SignatureAlgorithm = "ECDSA-secp256k1"
)

// IsValid checks if the signature algorithm is valid.
func (a SignatureAlgorithm) IsValid() bool {
	return a == AlgorithmEdDSA ||
		a == AlgorithmES256K ||
		a == AlgorithmECDSA ||
		a == AlgorithmECDSASecp256k1
}

// SignatureData contains signature information for message verification.
type SignatureData struct {
	// Algorithm is the signature algorithm used.
	Algorithm SignatureAlgorithm `json:"algorithm"`
	// KeyID is the identifier of the key used for signing.
	KeyID string `json:"keyId"`
	// Signature is the actual signature bytes.
	Signature []byte `json:"signature"`
	// SignedFields lists which fields were included in the signature.
	SignedFields []string `json:"signedFields"`
}

// Validate validates the signature data.
func (s *SignatureData) Validate() error {
	if !s.Algorithm.IsValid() {
		return fmt.Errorf("invalid signature algorithm: %s", s.Algorithm)
	}
	if s.KeyID == "" {
		return fmt.Errorf("KeyID is required")
	}
	if len(s.Signature) == 0 {
		return fmt.Errorf("signature is required")
	}
	if len(s.SignedFields) == 0 {
		return fmt.Errorf("signedFields cannot be empty")
	}
	return nil
}

// SecurityMetadata contains SAGE security-related metadata for a message.
type SecurityMetadata struct {
	// Mode is the protocol mode (a2a, sage, auto).
	Mode ProtocolMode `json:"mode"`
	// AgentDID is the decentralized identifier of the agent.
	AgentDID string `json:"agentDid"`
	// Nonce is a one-time random value to prevent replay attacks.
	Nonce string `json:"nonce"`
	// Timestamp records when this message was generated.
	Timestamp time.Time `json:"timestamp"`
	// Sequence is an ever-increasing packet counter for message ordering.
	Sequence uint64 `json:"sequence"`
	// Signature contains signature data for verification.
	Signature *SignatureData `json:"signature,omitempty"`
}

// Validate validates the security metadata.
func (s *SecurityMetadata) Validate() error {
	if !s.Mode.IsValid() {
		return fmt.Errorf("invalid protocol mode: %s", s.Mode)
	}

	// For SAGE mode, require security fields
	if s.Mode == ProtocolModeSAGE {
		if s.AgentDID == "" {
			return fmt.Errorf("AgentDID required for SAGE mode")
		}
		if s.Nonce == "" {
			return fmt.Errorf("Nonce required for SAGE mode")
		}
		if s.Timestamp.IsZero() {
			return fmt.Errorf("Timestamp required for SAGE mode")
		}
	}

	// Validate signature if present
	if s.Signature != nil {
		if err := s.Signature.Validate(); err != nil {
			return fmt.Errorf("signature validation failed: %w", err)
		}
	}

	return nil
}
