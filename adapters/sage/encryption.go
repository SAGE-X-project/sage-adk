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
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// EncryptionManager handles encryption and decryption for SAGE transport.
type EncryptionManager struct {
	// No state needed - stateless operations
}

// NewEncryptionManager creates a new encryption manager.
func NewEncryptionManager() *EncryptionManager {
	return &EncryptionManager{}
}

// GenerateEphemeralKeyPair generates a new X25519 ephemeral key pair for HPKE.
func (em *EncryptionManager) GenerateEphemeralKeyPair() (privateKey *ecdh.PrivateKey, publicKey []byte, err error) {
	// Generate X25519 key pair
	privKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, errors.ErrOperationFailed.
			WithMessage("failed to generate X25519 key pair").
			WithDetail("error", err.Error())
	}

	pubKey := privKey.PublicKey()
	return privKey, pubKey.Bytes(), nil
}

// DeriveSharedSecret performs ECDH key agreement to derive a shared secret.
// This implements HPKE key agreement using X25519.
func (em *EncryptionManager) DeriveSharedSecret(privateKey *ecdh.PrivateKey, remotePubKeyBytes []byte) ([]byte, error) {
	// Parse remote public key
	remotePubKey, err := ecdh.X25519().NewPublicKey(remotePubKeyBytes)
	if err != nil {
		return nil, errors.ErrInvalidInput.
			WithMessage("invalid remote public key").
			WithDetail("error", err.Error())
	}

	// Perform ECDH
	sharedSecret, err := privateKey.ECDH(remotePubKey)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("ECDH key agreement failed").
			WithDetail("error", err.Error())
	}

	// Derive key material using HKDF
	// Context info for key derivation
	info := []byte("SAGE-HPKE-v1")

	// Use HKDF to derive a 32-byte key from the shared secret
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, info)
	derivedKey := make([]byte, 32)
	if _, err := hkdfReader.Read(derivedKey); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("HKDF key derivation failed").
			WithDetail("error", err.Error())
	}

	return derivedKey, nil
}

// GenerateSessionKey generates a random 32-byte session key for ChaCha20-Poly1305.
func (em *EncryptionManager) GenerateSessionKey() ([]byte, error) {
	key := make([]byte, chacha20poly1305.KeySize) // 32 bytes
	if _, err := rand.Read(key); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to generate session key").
			WithDetail("error", err.Error())
	}
	return key, nil
}

// EncryptWithSharedSecret encrypts data using the HPKE-derived shared secret.
// Used in Phase 2 (Request) to encrypt the payload.
func (em *EncryptionManager) EncryptWithSharedSecret(data interface{}, sharedSecret []byte) (*EncryptedPayload, error) {
	// Serialize data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to marshal data").
			WithDetail("error", err.Error())
	}

	// Create ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(sharedSecret)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to create cipher").
			WithDetail("error", err.Error())
	}

	// Generate random nonce (12 bytes for ChaCha20-Poly1305)
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to generate nonce").
			WithDetail("error", err.Error())
	}

	// Encrypt
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return &EncryptedPayload{
		Algorithm:  "ChaCha20-Poly1305",
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

// DecryptWithSharedSecret decrypts data using the HPKE-derived shared secret.
func (em *EncryptionManager) DecryptWithSharedSecret(payload *EncryptedPayload, sharedSecret []byte, target interface{}) error {
	if payload == nil {
		return errors.ErrInvalidInput.WithMessage("encrypted payload is nil")
	}

	// Decode ciphertext and nonce from base64
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return errors.ErrInvalidInput.
			WithMessage("failed to decode ciphertext").
			WithDetail("error", err.Error())
	}

	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return errors.ErrInvalidInput.
			WithMessage("failed to decode nonce").
			WithDetail("error", err.Error())
	}

	// Create ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(sharedSecret)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to create cipher").
			WithDetail("error", err.Error())
	}

	// Decrypt
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("decryption failed").
			WithDetail("error", err.Error())
	}

	// Unmarshal JSON
	if err := json.Unmarshal(plaintext, target); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to unmarshal decrypted data").
			WithDetail("error", err.Error())
	}

	return nil
}

// EncryptWithSessionKey encrypts data using the session key.
// Used in Phase 3 (Response), Phase 4 (Complete), and application messages.
func (em *EncryptionManager) EncryptWithSessionKey(data interface{}, sessionKey []byte) (*EncryptedPayload, error) {
	// Validate session key
	if len(sessionKey) != chacha20poly1305.KeySize {
		return nil, errors.ErrInvalidInput.
			WithMessage("invalid session key size").
			WithDetail("expected", fmt.Sprintf("%d bytes", chacha20poly1305.KeySize)).
			WithDetail("got", fmt.Sprintf("%d bytes", len(sessionKey)))
	}

	// Serialize data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to marshal data").
			WithDetail("error", err.Error())
	}

	// Create ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(sessionKey)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to create cipher").
			WithDetail("error", err.Error())
	}

	// Generate random nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to generate nonce").
			WithDetail("error", err.Error())
	}

	// Encrypt
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return &EncryptedPayload{
		Algorithm:  "ChaCha20-Poly1305",
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

// DecryptWithSessionKey decrypts data using the session key.
func (em *EncryptionManager) DecryptWithSessionKey(payload *EncryptedPayload, sessionKey []byte, target interface{}) error {
	if payload == nil {
		return errors.ErrInvalidInput.WithMessage("encrypted payload is nil")
	}

	// Validate session key
	if len(sessionKey) != chacha20poly1305.KeySize {
		return errors.ErrInvalidInput.
			WithMessage("invalid session key size").
			WithDetail("expected", fmt.Sprintf("%d bytes", chacha20poly1305.KeySize)).
			WithDetail("got", fmt.Sprintf("%d bytes", len(sessionKey)))
	}

	// Decode ciphertext and nonce from base64
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return errors.ErrInvalidInput.
			WithMessage("failed to decode ciphertext").
			WithDetail("error", err.Error())
	}

	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return errors.ErrInvalidInput.
			WithMessage("failed to decode nonce").
			WithDetail("error", err.Error())
	}

	// Create ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(sessionKey)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to create cipher").
			WithDetail("error", err.Error())
	}

	// Decrypt
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("decryption failed - invalid key or corrupted data").
			WithDetail("error", err.Error())
	}

	// Unmarshal JSON
	if err := json.Unmarshal(plaintext, target); err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to unmarshal decrypted data").
			WithDetail("error", err.Error())
	}

	return nil
}

// EncryptPayloadForPublicKey encrypts data for a recipient's public key using HPKE.
// This is used in Phase 2 where Bob encrypts for Alice's ephemeral public key.
func (em *EncryptionManager) EncryptPayloadForPublicKey(data interface{}, recipientPubKeyBytes []byte) (*EncryptedPayload, []byte, error) {
	// Generate ephemeral key pair for encryption
	ephemeralPriv, ephemeralPub, err := em.GenerateEphemeralKeyPair()
	if err != nil {
		return nil, nil, err
	}

	// Derive shared secret
	sharedSecret, err := em.DeriveSharedSecret(ephemeralPriv, recipientPubKeyBytes)
	if err != nil {
		return nil, nil, err
	}

	// Encrypt with shared secret
	encryptedPayload, err := em.EncryptWithSharedSecret(data, sharedSecret)
	if err != nil {
		return nil, nil, err
	}

	// Return encrypted payload and ephemeral public key
	// The recipient needs the ephemeral public key to derive the same shared secret
	encryptedPayload.Algorithm = "HPKE"
	return encryptedPayload, ephemeralPub, nil
}

// DecryptPayloadWithPrivateKey decrypts HPKE-encrypted data using recipient's private key.
func (em *EncryptionManager) DecryptPayloadWithPrivateKey(payload *EncryptedPayload, privateKey *ecdh.PrivateKey, senderEphemeralPubKey []byte, target interface{}) error {
	if payload == nil {
		return errors.ErrInvalidInput.WithMessage("encrypted payload is nil")
	}

	if privateKey == nil {
		return errors.ErrInvalidInput.WithMessage("private key is nil")
	}

	// Derive shared secret using sender's ephemeral public key
	sharedSecret, err := em.DeriveSharedSecret(privateKey, senderEphemeralPubKey)
	if err != nil {
		return err
	}

	// Decrypt with shared secret
	return em.DecryptWithSharedSecret(payload, sharedSecret, target)
}
