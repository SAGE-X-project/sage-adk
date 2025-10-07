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
	"time"
)

// HandshakePhase represents the current phase of the SAGE handshake.
type HandshakePhase string

const (
	// PhaseInvitation is the first phase where Alice initiates communication.
	PhaseInvitation HandshakePhase = "invitation"

	// PhaseRequest is the second phase where Bob responds with key agreement.
	PhaseRequest HandshakePhase = "request"

	// PhaseResponse is the third phase where Alice confirms and sends session key.
	PhaseResponse HandshakePhase = "response"

	// PhaseComplete is the final phase where Bob acknowledges session establishment.
	PhaseComplete HandshakePhase = "complete"
)

// SessionStatus represents the current status of a SAGE session.
type SessionStatus int

const (
	// SessionPending indicates session is being created.
	SessionPending SessionStatus = iota

	// SessionEstablishing indicates handshake is in progress.
	SessionEstablishing

	// SessionActive indicates session is ready for messages.
	SessionActive

	// SessionExpired indicates session has expired.
	SessionExpired

	// SessionClosed indicates session was explicitly closed.
	SessionClosed
)

// String returns the string representation of SessionStatus.
func (s SessionStatus) String() string {
	switch s {
	case SessionPending:
		return "pending"
	case SessionEstablishing:
		return "establishing"
	case SessionActive:
		return "active"
	case SessionExpired:
		return "expired"
	case SessionClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Session represents an active SAGE session between two agents.
type Session struct {
	// Session identification
	ID        string
	LocalDID  string
	RemoteDID string

	// Cryptographic material
	SessionKey   []byte // ChaCha20-Poly1305 key (32 bytes)
	SharedSecret []byte // HPKE shared secret

	// Timing
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastActive time.Time

	// Status
	Status SessionStatus

	// Handshake state
	LocalNonce      string
	RemoteNonce     string
	EphemeralKey    []byte // X25519 private key
	EphemeralPubKey []byte // X25519 public key

	// Statistics
	MessagesSent     int64
	MessagesReceived int64

	// Metadata
	Metadata map[string]interface{}
}

// IsActive returns true if the session is active and not expired.
func (s *Session) IsActive() bool {
	return s.Status == SessionActive && time.Now().Before(s.ExpiresAt)
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt) || s.Status == SessionExpired
}

// UpdateActivity updates the last activity timestamp.
func (s *Session) UpdateActivity() {
	s.LastActive = time.Now()
}

// HandshakeInvitation represents Phase 1 of the SAGE handshake.
type HandshakeInvitation struct {
	Phase                 HandshakePhase `json:"phase"`
	FromDID               string         `json:"from_did"`
	ToDID                 string         `json:"to_did"`
	Nonce                 string         `json:"nonce"`
	EphemeralPublicKey    string         `json:"ephemeral_public_key"`
	SupportedAlgorithms   []string       `json:"supported_algorithms"`
	Capabilities          []string       `json:"capabilities"`
	Timestamp             time.Time      `json:"timestamp"`
}

// HandshakeRequest represents Phase 2 of the SAGE handshake.
type HandshakeRequest struct {
	Phase              HandshakePhase     `json:"phase"`
	SessionID          string             `json:"session_id"`
	FromDID            string             `json:"from_did"`
	ToDID              string             `json:"to_did"`
	Nonce              string             `json:"nonce"`
	EphemeralPublicKey string             `json:"ephemeral_public_key"`
	EncryptedPayload   EncryptedPayload   `json:"encrypted_payload"`
	Signature          SignatureEnvelope  `json:"signature"`
	Timestamp          time.Time          `json:"timestamp"`
}

// HandshakeRequestPayload is the decrypted payload of Phase 2.
type HandshakeRequestPayload struct {
	InvitationNonce       string `json:"invitation_nonce"`
	ResponseNonce         string `json:"response_nonce"`
	SharedSecretProposal  string `json:"shared_secret_proposal"`
}

// HandshakeResponse represents Phase 3 of the SAGE handshake.
type HandshakeResponse struct {
	Phase            HandshakePhase     `json:"phase"`
	SessionID        string             `json:"session_id"`
	FromDID          string             `json:"from_did"`
	ToDID            string             `json:"to_did"`
	EncryptedPayload EncryptedPayload   `json:"encrypted_payload"`
	Signature        SignatureEnvelope  `json:"signature"`
	Timestamp        time.Time          `json:"timestamp"`
}

// HandshakeResponsePayload is the decrypted payload of Phase 3.
type HandshakeResponsePayload struct {
	RequestNonce string                 `json:"request_nonce"`
	SessionKey   string                 `json:"session_key"`
	Expiry       time.Time              `json:"expiry"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// HandshakeComplete represents Phase 4 of the SAGE handshake.
type HandshakeComplete struct {
	Phase            HandshakePhase     `json:"phase"`
	SessionID        string             `json:"session_id"`
	FromDID          string             `json:"from_did"`
	ToDID            string             `json:"to_did"`
	EncryptedPayload EncryptedPayload   `json:"encrypted_payload"`
	Signature        SignatureEnvelope  `json:"signature"`
	Timestamp        time.Time          `json:"timestamp"`
}

// HandshakeCompletePayload is the decrypted payload of Phase 4.
type HandshakeCompletePayload struct {
	Ack             string                 `json:"ack"`
	SessionMetadata map[string]interface{} `json:"session_metadata"`
}

// EncryptedPayload represents encrypted data with algorithm metadata.
type EncryptedPayload struct {
	Algorithm  string `json:"algorithm"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce,omitempty"`
}

// SignatureEnvelope represents an RFC 9421 signature.
type SignatureEnvelope struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Value     string `json:"value"`
}

// SecureMessage represents an encrypted and signed message after handshake.
type SecureMessage struct {
	MessageID        string            `json:"message_id"`
	SessionID        string            `json:"session_id"`
	FromDID          string            `json:"from_did"`
	ToDID            string            `json:"to_did"`
	EncryptedPayload EncryptedPayload  `json:"encrypted_payload"`
	Signature        SignatureEnvelope `json:"signature"`
	Timestamp        time.Time         `json:"timestamp"`
}

// TransportConfig contains configuration for SAGE transport.
type TransportConfig struct {
	// Agent identity
	LocalDID   string
	RemoteDID  string

	// Blockchain configuration
	Network     string
	RPCEndpoint string

	// Cryptographic keys
	PrivateKey []byte // Ed25519 private key for signing

	// Session configuration
	SessionTTL      time.Duration
	MaxMessageSize  int64

	// Timeouts
	HandshakeTimeout time.Duration
	MessageTimeout   time.Duration

	// Security
	MaxClockSkew     time.Duration
	NonceCache      int // Number of nonces to cache for replay protection
}

// DefaultTransportConfig returns default configuration values.
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		SessionTTL:       time.Hour,
		MaxMessageSize:   10 * 1024 * 1024, // 10MB
		HandshakeTimeout: 30 * time.Second,
		MessageTimeout:   10 * time.Second,
		MaxClockSkew:     5 * time.Minute,
		NonceCache:       1000,
	}
}
