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
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// TransportManager manages SAGE protocol transport layer.
// It orchestrates handshakes, message encryption/decryption, and session lifecycle.
type TransportManager struct {
	handshakeManager  *HandshakeManager
	sessionManager    *SessionManager
	encryptionManager *EncryptionManager
	signingManager    *SigningManager

	// Agent identity
	localDID   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	// Configuration
	config *TransportConfig

	// Message handler
	messageHandler MessageHandler

	// Active handshakes tracking
	activeHandshakes map[string]*HandshakeState
	mu               sync.RWMutex
}

// HandshakeState tracks the state of an ongoing handshake.
type HandshakeState struct {
	Phase     HandshakePhase
	Session   *Session
	StartedAt time.Time
	UpdatedAt time.Time
}

// MessageHandler is called when an application message is received.
type MessageHandler func(ctx context.Context, fromDID string, payload []byte) error

// NewTransportManager creates a new transport manager.
func NewTransportManager(
	localDID string,
	privateKey ed25519.PrivateKey,
	config *TransportConfig,
) *TransportManager {
	if config == nil {
		config = DefaultTransportConfig()
	}

	// Use default cleanup interval of 5 minutes
	cleanupInterval := 5 * time.Minute
	sessionManager := NewSessionManager(config.SessionTTL, cleanupInterval)
	encryptionManager := NewEncryptionManager()
	signingManager := NewSigningManager()

	handshakeManager := NewHandshakeManager(
		sessionManager,
		encryptionManager,
		signingManager,
		localDID,
		privateKey,
		config,
	)

	return &TransportManager{
		handshakeManager:  handshakeManager,
		sessionManager:    sessionManager,
		encryptionManager: encryptionManager,
		signingManager:    signingManager,
		localDID:          localDID,
		privateKey:        privateKey,
		publicKey:         privateKey.Public().(ed25519.PublicKey),
		config:            config,
		activeHandshakes:  make(map[string]*HandshakeState),
	}
}

// SetMessageHandler sets the handler for incoming application messages.
func (tm *TransportManager) SetMessageHandler(handler MessageHandler) {
	tm.messageHandler = handler
}

// Connect initiates a connection to a remote agent.
// This starts the handshake process (Phase 1: Invitation).
func (tm *TransportManager) Connect(ctx context.Context, remoteDID string) (*HandshakeInvitation, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if already connected
	session, err := tm.sessionManager.GetByDID(remoteDID)
	if err == nil && session.IsActive() {
		return nil, errors.ErrOperationFailed.
			WithMessage("already connected to remote agent").
			WithDetail("remote_did", remoteDID)
	}

	// Check if handshake already in progress
	if _, exists := tm.activeHandshakes[remoteDID]; exists {
		return nil, errors.ErrOperationFailed.
			WithMessage("handshake already in progress").
			WithDetail("remote_did", remoteDID)
	}

	// Initiate handshake
	invitation, session, err := tm.handshakeManager.InitiateHandshake(ctx, remoteDID)
	if err != nil {
		return nil, err
	}

	// Track handshake state
	tm.activeHandshakes[remoteDID] = &HandshakeState{
		Phase:     PhaseInvitation,
		Session:   session,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return invitation, nil
}

// HandleInvitation processes an incoming invitation (initiator is the remote agent).
// Bob calls this when receiving Alice's invitation.
func (tm *TransportManager) HandleInvitation(ctx context.Context, invitation *HandshakeInvitation) (*HandshakeRequest, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Process invitation
	request, session, err := tm.handshakeManager.ProcessInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}

	// Track handshake state
	tm.activeHandshakes[invitation.FromDID] = &HandshakeState{
		Phase:     PhaseRequest,
		Session:   session,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return request, nil
}

// HandleRequest processes an incoming request.
// Alice calls this when receiving Bob's request.
func (tm *TransportManager) HandleRequest(ctx context.Context, request *HandshakeRequest, bobPublicKey ed25519.PublicKey) (*HandshakeResponse, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get handshake state
	state, exists := tm.activeHandshakes[request.FromDID]
	if !exists {
		return nil, errors.ErrOperationFailed.
			WithMessage("no active handshake found").
			WithDetail("remote_did", request.FromDID)
	}

	if state.Phase != PhaseInvitation {
		return nil, errors.ErrOperationFailed.
			WithMessage("unexpected handshake phase").
			WithDetail("expected", string(PhaseInvitation)).
			WithDetail("got", string(state.Phase))
	}

	// Process request
	response, err := tm.handshakeManager.ProcessRequest(ctx, request, state.Session, bobPublicKey)
	if err != nil {
		return nil, err
	}

	// Update handshake state
	state.Phase = PhaseResponse
	state.UpdatedAt = time.Now()

	return response, nil
}

// HandleResponse processes an incoming response.
// Bob calls this when receiving Alice's response.
func (tm *TransportManager) HandleResponse(ctx context.Context, response *HandshakeResponse, alicePublicKey ed25519.PublicKey) (*HandshakeComplete, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get handshake state
	state, exists := tm.activeHandshakes[response.FromDID]
	if !exists {
		return nil, errors.ErrOperationFailed.
			WithMessage("no active handshake found").
			WithDetail("remote_did", response.FromDID)
	}

	if state.Phase != PhaseRequest {
		return nil, errors.ErrOperationFailed.
			WithMessage("unexpected handshake phase").
			WithDetail("expected", string(PhaseRequest)).
			WithDetail("got", string(state.Phase))
	}

	// Process response
	complete, err := tm.handshakeManager.ProcessResponse(ctx, response, state.Session, alicePublicKey)
	if err != nil {
		return nil, err
	}

	// Update handshake state
	state.Phase = PhaseComplete
	state.UpdatedAt = time.Now()

	return complete, nil
}

// HandleComplete processes an incoming complete message.
// Alice calls this when receiving Bob's complete message.
func (tm *TransportManager) HandleComplete(ctx context.Context, complete *HandshakeComplete, bobPublicKey ed25519.PublicKey) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get handshake state
	state, exists := tm.activeHandshakes[complete.FromDID]
	if !exists {
		return errors.ErrOperationFailed.
			WithMessage("no active handshake found").
			WithDetail("remote_did", complete.FromDID)
	}

	if state.Phase != PhaseResponse {
		return errors.ErrOperationFailed.
			WithMessage("unexpected handshake phase").
			WithDetail("expected", string(PhaseResponse)).
			WithDetail("got", string(state.Phase))
	}

	// Process complete
	if err := tm.handshakeManager.ProcessComplete(ctx, complete, state.Session, bobPublicKey); err != nil {
		return err
	}

	// Remove from active handshakes (now complete)
	delete(tm.activeHandshakes, complete.FromDID)

	return nil
}

// SendMessage sends an encrypted message to a remote agent.
// The session must be established first via Connect/Handle* methods.
func (tm *TransportManager) SendMessage(ctx context.Context, remoteDID string, payload interface{}) (*ApplicationMessage, error) {
	// Get active session
	session, err := tm.sessionManager.GetByDID(remoteDID)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("no active session found").
			WithDetail("remote_did", remoteDID).
			WithDetail("error", err.Error())
	}

	if !session.IsActive() {
		return nil, errors.ErrOperationFailed.
			WithMessage("session is not active").
			WithDetail("remote_did", remoteDID).
			WithDetail("status", session.Status)
	}

	// Encrypt payload with session key
	encryptedPayload, err := tm.encryptionManager.EncryptWithSessionKey(payload, session.SessionKey)
	if err != nil {
		return nil, err
	}

	// Create application message
	message := &ApplicationMessage{
		FromDID:          tm.localDID,
		ToDID:            remoteDID,
		SessionID:        session.ID,
		EncryptedPayload: *encryptedPayload,
		Timestamp:        time.Now(),
	}

	// Sign message
	keyID := tm.localDID + "#key-1"
	signature, err := tm.signingManager.SignMessage(message, tm.privateKey, keyID)
	if err != nil {
		return nil, err
	}
	message.Signature = *signature

	return message, nil
}

// ReceiveMessage processes an incoming encrypted message.
func (tm *TransportManager) ReceiveMessage(ctx context.Context, message *ApplicationMessage, senderPublicKey ed25519.PublicKey) error {
	// Verify signature
	if err := tm.signingManager.VerifySignature(message, &message.Signature, senderPublicKey); err != nil {
		return errors.ErrSignatureInvalid.
			WithMessage("message signature verification failed").
			WithDetail("error", err.Error())
	}

	// Validate timestamp
	if err := tm.signingManager.ValidateTimestamp(message.Timestamp, tm.config.MaxClockSkew); err != nil {
		return err
	}

	// Get session
	session, err := tm.sessionManager.GetByDID(message.FromDID)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("no active session found").
			WithDetail("from_did", message.FromDID).
			WithDetail("error", err.Error())
	}

	if !session.IsActive() {
		return errors.ErrOperationFailed.
			WithMessage("session is not active").
			WithDetail("from_did", message.FromDID).
			WithDetail("status", session.Status)
	}

	// Decrypt payload
	var payload map[string]interface{}
	if err := tm.encryptionManager.DecryptWithSessionKey(&message.EncryptedPayload, session.SessionKey, &payload); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to decrypt message").
			WithDetail("error", err.Error())
	}

	// Convert payload to JSON bytes for handler
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to marshal payload").
			WithDetail("error", err.Error())
	}

	// Call message handler if set
	if tm.messageHandler != nil {
		return tm.messageHandler(ctx, message.FromDID, payloadBytes)
	}

	return nil
}

// Disconnect closes a connection to a remote agent.
func (tm *TransportManager) Disconnect(ctx context.Context, remoteDID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Remove from active handshakes if exists
	delete(tm.activeHandshakes, remoteDID)

	// Get session
	session, err := tm.sessionManager.GetByDID(remoteDID)
	if err != nil {
		// Session doesn't exist, already disconnected
		return nil
	}

	// Close session
	return tm.sessionManager.Delete(session.ID)
}

// GetSession returns the session for a remote DID.
func (tm *TransportManager) GetSession(remoteDID string) (*Session, error) {
	return tm.sessionManager.GetByDID(remoteDID)
}

// ListSessions returns all active sessions.
func (tm *TransportManager) ListSessions() []*Session {
	return tm.sessionManager.List()
}

// Close shuts down the transport manager.
func (tm *TransportManager) Close() error {
	// Close all active sessions
	sessions := tm.sessionManager.List()
	for _, session := range sessions {
		tm.sessionManager.Delete(session.ID)
	}
	return nil
}

// ApplicationMessage represents an encrypted application-level message.
type ApplicationMessage struct {
	FromDID          string             `json:"from_did"`
	ToDID            string             `json:"to_did"`
	SessionID        string             `json:"session_id"`
	EncryptedPayload EncryptedPayload   `json:"encrypted_payload"`
	Signature        SignatureEnvelope  `json:"signature"`
	Timestamp        time.Time          `json:"timestamp"`
}

// MessageEnvelope wraps any SAGE protocol message for transport.
type MessageEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// WrapMessage wraps a message in an envelope for transport.
func WrapMessage(messageType string, payload interface{}) (*MessageEnvelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to marshal payload").
			WithDetail("error", err.Error())
	}

	return &MessageEnvelope{
		Type:    messageType,
		Payload: payloadBytes,
	}, nil
}

// UnwrapMessage unwraps a message from an envelope.
func UnwrapMessage(envelope *MessageEnvelope, target interface{}) error {
	if err := json.Unmarshal(envelope.Payload, target); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to unmarshal payload").
			WithDetail("error", err.Error())
	}
	return nil
}

// SerializeMessage serializes a message to JSON bytes.
func SerializeMessage(message interface{}) ([]byte, error) {
	return json.Marshal(message)
}

// DeserializeMessage deserializes JSON bytes to a message.
func DeserializeMessage(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

// EncodeMessage encodes a message to base64 string.
func EncodeMessage(message interface{}) (string, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeMessage decodes a base64 string to a message.
func DecodeMessage(encoded string, target interface{}) error {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
