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
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// HandshakeManager orchestrates the 4-phase SAGE handshake protocol.
type HandshakeManager struct {
	sessionManager    *SessionManager
	encryptionManager *EncryptionManager
	signingManager    *SigningManager

	// Agent identity
	localDID   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	// Configuration
	config *TransportConfig
}

// NewHandshakeManager creates a new handshake manager.
func NewHandshakeManager(
	sessionManager *SessionManager,
	encryptionManager *EncryptionManager,
	signingManager *SigningManager,
	localDID string,
	privateKey ed25519.PrivateKey,
	config *TransportConfig,
) *HandshakeManager {
	return &HandshakeManager{
		sessionManager:    sessionManager,
		encryptionManager: encryptionManager,
		signingManager:    signingManager,
		localDID:          localDID,
		privateKey:        privateKey,
		publicKey:         privateKey.Public().(ed25519.PublicKey),
		config:            config,
	}
}

// InitiateHandshake starts the handshake process (Phase 1: Invitation).
// Alice calls this to initiate communication with Bob.
func (hm *HandshakeManager) InitiateHandshake(ctx context.Context, remoteDID string) (*HandshakeInvitation, *Session, error) {
	// Create session
	session, err := hm.sessionManager.Create(hm.localDID, remoteDID)
	if err != nil {
		return nil, nil, errors.ErrOperationFailed.
			WithMessage("failed to create session").
			WithDetail("error", err.Error())
	}

	// Mark session as establishing
	session.Status = SessionEstablishing

	// Generate ephemeral key pair for HPKE
	ephemeralPrivKey, ephemeralPubKey, err := hm.encryptionManager.GenerateEphemeralKeyPair()
	if err != nil {
		return nil, nil, err
	}

	// Store ephemeral key in session
	session.EphemeralKey = ephemeralPrivKey.Bytes()
	session.EphemeralPubKey = ephemeralPubKey

	// Generate nonce
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, nil, err
	}
	session.LocalNonce = nonce

	// Update session
	if err := hm.sessionManager.Update(session); err != nil {
		return nil, nil, err
	}

	// Create invitation
	invitation := &HandshakeInvitation{
		Phase:              PhaseInvitation,
		FromDID:            hm.localDID,
		ToDID:              remoteDID,
		Nonce:              nonce,
		EphemeralPublicKey: base64.StdEncoding.EncodeToString(ephemeralPubKey),
		SupportedAlgorithms: []string{"EdDSA", "ECDSA-secp256k1"},
		Capabilities:        []string{"messaging"},
		Timestamp:          time.Now(),
	}

	return invitation, session, nil
}

// ProcessInvitation processes an invitation (Phase 1) and creates a request (Phase 2).
// Bob calls this when receiving Alice's invitation.
func (hm *HandshakeManager) ProcessInvitation(ctx context.Context, invitation *HandshakeInvitation) (*HandshakeRequest, *Session, error) {
	// Validate invitation
	if err := hm.ValidateInvitation(invitation); err != nil {
		return nil, nil, err
	}

	// Create session
	session, err := hm.sessionManager.Create(hm.localDID, invitation.FromDID)
	if err != nil {
		return nil, nil, err
	}

	session.Status = SessionEstablishing
	session.RemoteNonce = invitation.Nonce

	// Decode Alice's ephemeral public key
	aliceEphemeralPubKey, err := base64.StdEncoding.DecodeString(invitation.EphemeralPublicKey)
	if err != nil {
		return nil, nil, errors.ErrInvalidInput.
			WithMessage("failed to decode ephemeral public key").
			WithDetail("error", err.Error())
	}

	// Generate Bob's ephemeral key pair
	bobEphemeralPrivKey, bobEphemeralPubKey, err := hm.encryptionManager.GenerateEphemeralKeyPair()
	if err != nil {
		return nil, nil, err
	}

	session.EphemeralKey = bobEphemeralPrivKey.Bytes()
	session.EphemeralPubKey = bobEphemeralPubKey

	// Derive shared secret using HPKE
	sharedSecret, err := hm.encryptionManager.DeriveSharedSecret(bobEphemeralPrivKey, aliceEphemeralPubKey)
	if err != nil {
		return nil, nil, err
	}
	session.SharedSecret = sharedSecret

	// Generate Bob's nonce
	bobNonce, err := GenerateNonce()
	if err != nil {
		return nil, nil, err
	}
	session.LocalNonce = bobNonce

	// Update session
	if err := hm.sessionManager.Update(session); err != nil {
		return nil, nil, err
	}

	// Create request payload
	requestPayload := &HandshakeRequestPayload{
		InvitationNonce:      invitation.Nonce,
		ResponseNonce:        bobNonce,
		SharedSecretProposal: base64.StdEncoding.EncodeToString(sharedSecret),
	}

	// Encrypt payload with shared secret
	encryptedPayload, err := hm.encryptionManager.EncryptWithSharedSecret(requestPayload, sharedSecret)
	if err != nil {
		return nil, nil, err
	}

	// Create request
	request := &HandshakeRequest{
		Phase:              PhaseRequest,
		SessionID:          session.ID,
		FromDID:            hm.localDID,
		ToDID:              invitation.FromDID,
		Nonce:              bobNonce,
		EphemeralPublicKey: base64.StdEncoding.EncodeToString(bobEphemeralPubKey),
		EncryptedPayload:   *encryptedPayload,
		Timestamp:          time.Now(),
	}

	// Sign request
	keyID := hm.localDID + "#key-1"
	signature, err := hm.signingManager.SignMessage(request, hm.privateKey, keyID)
	if err != nil {
		return nil, nil, err
	}
	request.Signature = *signature

	return request, session, nil
}

// ProcessRequest processes a request (Phase 2) and creates a response (Phase 3).
// Alice calls this when receiving Bob's request.
func (hm *HandshakeManager) ProcessRequest(ctx context.Context, request *HandshakeRequest, session *Session, bobPublicKey ed25519.PublicKey) (*HandshakeResponse, error) {
	// Verify signature
	if err := hm.signingManager.VerifySignature(request, &request.Signature, bobPublicKey); err != nil {
		return nil, errors.ErrSignatureInvalid.
			WithMessage("request signature verification failed").
			WithDetail("error", err.Error())
	}

	// Validate request
	if err := hm.ValidateRequest(request); err != nil {
		return nil, err
	}

	// Note: Alice's session ID and Bob's session ID are different
	// Alice will use request.SessionID for the response so Bob recognizes it

	// Decode Bob's ephemeral public key
	bobEphemeralPubKey, err := base64.StdEncoding.DecodeString(request.EphemeralPublicKey)
	if err != nil {
		return nil, errors.ErrInvalidInput.
			WithMessage("failed to decode ephemeral public key").
			WithDetail("error", err.Error())
	}

	// Derive shared secret using Alice's ephemeral private key
	aliceEphemeralPrivKey, err := hm.reconstructPrivateKey(session.EphemeralKey)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := hm.encryptionManager.DeriveSharedSecret(aliceEphemeralPrivKey, bobEphemeralPubKey)
	if err != nil {
		return nil, err
	}
	session.SharedSecret = sharedSecret

	// Decrypt request payload
	var requestPayload HandshakeRequestPayload
	if err := hm.encryptionManager.DecryptWithSharedSecret(&request.EncryptedPayload, sharedSecret, &requestPayload); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to decrypt request payload").
			WithDetail("error", err.Error())
	}

	// Verify invitation nonce matches
	if requestPayload.InvitationNonce != session.LocalNonce {
		return nil, errors.ErrOperationFailed.
			WithMessage("invitation nonce mismatch")
	}

	session.RemoteNonce = requestPayload.ResponseNonce

	// Generate session key
	sessionKey, err := hm.encryptionManager.GenerateSessionKey()
	if err != nil {
		return nil, err
	}
	session.SessionKey = sessionKey

	// Set expiry
	session.ExpiresAt = time.Now().Add(hm.config.SessionTTL)

	// Update session
	if err := hm.sessionManager.Update(session); err != nil {
		return nil, err
	}

	// Create response payload
	responsePayload := &HandshakeResponsePayload{
		RequestNonce: requestPayload.ResponseNonce,
		SessionKey: base64.StdEncoding.EncodeToString(sessionKey),
		Expiry:     session.ExpiresAt,
	}

	// Encrypt with shared secret
	encryptedPayload, err := hm.encryptionManager.EncryptWithSharedSecret(responsePayload, sharedSecret)
	if err != nil {
		return nil, err
	}

	// Create response
	// Use Bob's session ID so he can match it to his session
	response := &HandshakeResponse{
		Phase:            PhaseResponse,
		SessionID:        request.SessionID,
		FromDID:          hm.localDID,
		ToDID:            request.FromDID,
		EncryptedPayload: *encryptedPayload,
		Timestamp:        time.Now(),
	}

	// Sign response
	keyID := hm.localDID + "#key-1"
	signature, err := hm.signingManager.SignMessage(response, hm.privateKey, keyID)
	if err != nil {
		return nil, err
	}
	response.Signature = *signature

	return response, nil
}

// ProcessResponse processes a response (Phase 3) and creates a complete message (Phase 4).
// Bob calls this when receiving Alice's response.
func (hm *HandshakeManager) ProcessResponse(ctx context.Context, response *HandshakeResponse, session *Session, alicePublicKey ed25519.PublicKey) (*HandshakeComplete, error) {
	// Verify signature
	if err := hm.signingManager.VerifySignature(response, &response.Signature, alicePublicKey); err != nil {
		return nil, errors.ErrSignatureInvalid.
			WithMessage("response signature verification failed").
			WithDetail("error", err.Error())
	}

	// Validate response
	if err := hm.validateResponse(response, session); err != nil {
		return nil, err
	}

	// Decrypt response payload with shared secret
	var responsePayload HandshakeResponsePayload
	if err := hm.encryptionManager.DecryptWithSharedSecret(&response.EncryptedPayload, session.SharedSecret, &responsePayload); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to decrypt response payload").
			WithDetail("error", err.Error())
	}

	// Verify request nonce matches
	if responsePayload.RequestNonce != session.LocalNonce {
		return nil, errors.ErrOperationFailed.
			WithMessage("request nonce mismatch")
	}

	// Decode session key
	sessionKey, err := base64.StdEncoding.DecodeString(responsePayload.SessionKey)
	if err != nil {
		return nil, errors.ErrInvalidInput.
			WithMessage("failed to decode session key").
			WithDetail("error", err.Error())
	}
	session.SessionKey = sessionKey
	session.ExpiresAt = responsePayload.Expiry

	// Update session
	if err := hm.sessionManager.Update(session); err != nil {
		return nil, err
	}

	// Create complete payload
	completePayload := &HandshakeCompletePayload{
		Ack: "session_established",
		SessionMetadata: map[string]interface{}{
			"protocol_version": "1.0.0",
		},
	}

	// Encrypt with session key
	encryptedPayload, err := hm.encryptionManager.EncryptWithSessionKey(completePayload, sessionKey)
	if err != nil {
		return nil, err
	}

	// Create complete message
	complete := &HandshakeComplete{
		Phase:            PhaseComplete,
		SessionID:        session.ID,
		FromDID:          hm.localDID,
		ToDID:            response.FromDID,
		EncryptedPayload: *encryptedPayload,
		Timestamp:        time.Now(),
	}

	// Sign complete
	keyID := hm.localDID + "#key-1"
	signature, err := hm.signingManager.SignMessage(complete, hm.privateKey, keyID)
	if err != nil {
		return nil, err
	}
	complete.Signature = *signature

	// Activate Bob's session (Bob has the session key now)
	session.Status = SessionActive
	if err := hm.sessionManager.Update(session); err != nil {
		return nil, err
	}

	return complete, nil
}

// ProcessComplete processes the complete message (Phase 4) and activates the session.
// Alice calls this when receiving Bob's complete message.
func (hm *HandshakeManager) ProcessComplete(ctx context.Context, complete *HandshakeComplete, session *Session, bobPublicKey ed25519.PublicKey) error {
	// Verify signature
	if err := hm.signingManager.VerifySignature(complete, &complete.Signature, bobPublicKey); err != nil {
		return errors.ErrSignatureInvalid.
			WithMessage("complete signature verification failed").
			WithDetail("error", err.Error())
	}

	// Validate complete
	if err := hm.validateComplete(complete, session); err != nil {
		return err
	}

	// Decrypt complete payload with session key
	var completePayload HandshakeCompletePayload
	if err := hm.encryptionManager.DecryptWithSessionKey(&complete.EncryptedPayload, session.SessionKey, &completePayload); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to decrypt complete payload").
			WithDetail("error", err.Error())
	}

	// Verify ack
	if completePayload.Ack != "session_established" {
		return errors.ErrOperationFailed.
			WithMessage("invalid ack in complete message")
	}

	// Activate session
	session.Status = SessionActive
	if err := hm.sessionManager.Update(session); err != nil {
		return err
	}

	return nil
}

// ValidateInvitation validates Phase 1 invitation.
func (hm *HandshakeManager) ValidateInvitation(invitation *HandshakeInvitation) error {
	if invitation == nil {
		return errors.ErrInvalidInput.WithMessage("invitation is nil")
	}

	if invitation.Phase != PhaseInvitation {
		return errors.ErrInvalidInput.WithMessage("invalid phase for invitation")
	}

	if invitation.FromDID == "" || invitation.ToDID == "" {
		return errors.ErrInvalidInput.WithMessage("missing DID in invitation")
	}

	if invitation.Nonce == "" {
		return errors.ErrInvalidInput.WithMessage("missing nonce in invitation")
	}

	if invitation.EphemeralPublicKey == "" {
		return errors.ErrInvalidInput.WithMessage("missing ephemeral public key")
	}

	// Validate timestamp
	if err := hm.signingManager.ValidateTimestamp(invitation.Timestamp, hm.config.MaxClockSkew); err != nil {
		return err
	}

	return nil
}

// ValidateRequest validates Phase 2 request.
func (hm *HandshakeManager) ValidateRequest(request *HandshakeRequest) error {
	if request == nil {
		return errors.ErrInvalidInput.WithMessage("request is nil")
	}

	if request.Phase != PhaseRequest {
		return errors.ErrInvalidInput.WithMessage("invalid phase for request")
	}

	if request.SessionID == "" {
		return errors.ErrInvalidInput.WithMessage("missing session ID")
	}

	if request.Nonce == "" {
		return errors.ErrInvalidInput.WithMessage("missing nonce in request")
	}

	// Validate timestamp
	if err := hm.signingManager.ValidateTimestamp(request.Timestamp, hm.config.MaxClockSkew); err != nil {
		return err
	}

	return nil
}

// validateRequestSession validates Phase 2 request against session.
func (hm *HandshakeManager) validateRequestSession(request *HandshakeRequest, session *Session) error {
	if request.SessionID != session.ID {
		return errors.ErrInvalidInput.WithMessage("session ID mismatch")
	}
	return nil
}

// validateResponse validates Phase 3 response.
func (hm *HandshakeManager) validateResponse(response *HandshakeResponse, session *Session) error {
	if response.Phase != PhaseResponse {
		return errors.ErrInvalidInput.WithMessage("invalid phase for response")
	}

	// Validate session ID matches (Bob's session ID should match)
	if response.SessionID != session.ID {
		return errors.ErrInvalidInput.WithMessage("session ID mismatch")
	}

	// Validate timestamp
	if err := hm.signingManager.ValidateTimestamp(response.Timestamp, hm.config.MaxClockSkew); err != nil {
		return err
	}

	return nil
}

// validateComplete validates Phase 4 complete.
func (hm *HandshakeManager) validateComplete(complete *HandshakeComplete, session *Session) error {
	if complete.Phase != PhaseComplete {
		return errors.ErrInvalidInput.WithMessage("invalid phase for complete")
	}

	// Note: We don't validate session ID here because Alice and Bob have different session IDs
	// Alice uses Bob's session ID in messages to Bob, and vice versa

	// Validate timestamp
	if err := hm.signingManager.ValidateTimestamp(complete.Timestamp, hm.config.MaxClockSkew); err != nil {
		return err
	}

	return nil
}

// reconstructPrivateKey reconstructs an ECDH private key from bytes.
func (hm *HandshakeManager) reconstructPrivateKey(keyBytes []byte) (*ecdh.PrivateKey, error) {
	// Import the key bytes as an X25519 private key
	privKey, err := ecdh.X25519().NewPrivateKey(keyBytes)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to reconstruct private key").
			WithDetail("error", err.Error())
	}
	return privKey, nil
}
