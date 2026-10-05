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
	"fmt"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
)

// KeyManager composes the public SAGE key, format, and storage APIs.
type KeyManager struct {
	storage crypto.KeyStorage
}

// NewKeyManager creates a new key manager using sage crypto library.
func NewKeyManager() *KeyManager {
	return &KeyManager{
		storage: storage.NewMemoryKeyStorage(),
	}
}

// NewKeyManagerWithStorage creates a new key manager with custom storage backend.
func NewKeyManagerWithStorage(storage crypto.KeyStorage) *KeyManager {
	if storage == nil {
		return NewKeyManager()
	}
	return &KeyManager{
		storage: storage,
	}
}

// Generate creates a new Ed25519 key pair using sage crypto library.
// Ed25519 is the default key type for SAGE agents.
func (km *KeyManager) Generate() (crypto.KeyPair, error) {
	return km.GenerateWithType(crypto.KeyTypeEd25519)
}

// GenerateWithType creates a new key pair of the specified type.
func (km *KeyManager) GenerateWithType(keyType crypto.KeyType) (crypto.KeyPair, error) {
	switch keyType {
	case crypto.KeyTypeEd25519:
		return keys.GenerateEd25519KeyPair()
	case crypto.KeyTypeSecp256k1:
		return keys.GenerateSecp256k1KeyPair()
	case crypto.KeyTypeP256:
		return keys.GenerateP256KeyPair()
	case crypto.KeyTypeX25519:
		return keys.GenerateX25519KeyPair()
	default:
		return nil, crypto.ErrInvalidKeyType
	}
}

// LoadFromFile loads a key pair from a file using sage crypto formats.
// Supports both PEM and JWK formats. The format is auto-detected.
func (km *KeyManager) LoadFromFile(path string) (crypto.KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Try PEM format first (more common for private keys)
	keyPair, err := km.ImportKeyPair(data, crypto.KeyFormatPEM)
	if err == nil {
		return keyPair, nil
	}

	// Try JWK format
	keyPair, err = km.ImportKeyPair(data, crypto.KeyFormatJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to import key (tried PEM and JWK formats): %w", err)
	}

	return keyPair, nil
}

// LoadFromFileWithFormat loads a key pair from a file with explicit format.
func (km *KeyManager) LoadFromFileWithFormat(path string, format crypto.KeyFormat) (crypto.KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	keyPair, err := km.ImportKeyPair(data, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import key: %w", err)
	}

	return keyPair, nil
}

// SaveToFile saves a key pair to a file using PEM format.
// PEM is the default format for maximum compatibility.
func (km *KeyManager) SaveToFile(keyPair crypto.KeyPair, path string) error {
	return km.SaveToFileWithFormat(keyPair, path, crypto.KeyFormatPEM)
}

// SaveToFileWithFormat saves a key pair to a file with explicit format.
func (km *KeyManager) SaveToFileWithFormat(keyPair crypto.KeyPair, path string, format crypto.KeyFormat) error {
	data, err := km.ExportKeyPair(keyPair, format)
	if err != nil {
		return fmt.Errorf("failed to export key: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// Store stores a key pair in the manager's storage backend.
func (km *KeyManager) Store(keyPair crypto.KeyPair) error {
	return km.storage.Store(keyPair.ID(), keyPair)
}

// Load loads a key pair by ID from the manager's storage backend.
func (km *KeyManager) Load(id string) (crypto.KeyPair, error) {
	return km.storage.Load(id)
}

// Delete deletes a key pair by ID from the manager's storage backend.
func (km *KeyManager) Delete(id string) error {
	return km.storage.Delete(id)
}

// List lists all stored key pair IDs.
func (km *KeyManager) List() ([]string, error) {
	return km.storage.List()
}

// ExtractEd25519PrivateKey extracts the Ed25519 private key from a KeyPair.
// This is useful for integration with transport layer that needs raw Ed25519 keys.
func (km *KeyManager) ExtractEd25519PrivateKey(keyPair crypto.KeyPair) (ed25519.PrivateKey, error) {
	if keyPair.Type() != crypto.KeyTypeEd25519 {
		return nil, fmt.Errorf("key pair is not Ed25519 (got %s)", keyPair.Type())
	}

	privateKey := keyPair.PrivateKey()
	ed25519Key, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("failed to convert private key to Ed25519")
	}

	return ed25519Key, nil
}

// ExtractEd25519PublicKey extracts the Ed25519 public key from a KeyPair.
func (km *KeyManager) ExtractEd25519PublicKey(keyPair crypto.KeyPair) (ed25519.PublicKey, error) {
	if keyPair.Type() != crypto.KeyTypeEd25519 {
		return nil, fmt.Errorf("key pair is not Ed25519 (got %s)", keyPair.Type())
	}

	publicKey := keyPair.PublicKey()
	ed25519Key, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to convert public key to Ed25519")
	}

	return ed25519Key, nil
}

// Manager returns the ADK key operations facade.
// Deprecated: use the KeyManager methods directly. SAGE no longer exports crypto.Manager.
func (km *KeyManager) Manager() *KeyManager { return km }

// ImportKeyPair imports a key using a public SAGE format implementation.
func (km *KeyManager) ImportKeyPair(data []byte, format crypto.KeyFormat) (crypto.KeyPair, error) {
	switch format {
	case crypto.KeyFormatPEM:
		return formats.NewPEMImporter().Import(data, format)
	case crypto.KeyFormatJWK:
		return formats.NewJWKImporter().Import(data, format)
	default:
		return nil, crypto.ErrInvalidKeyFormat
	}
}

// ExportKeyPair exports a key using a public SAGE format implementation.
func (km *KeyManager) ExportKeyPair(key crypto.KeyPair, format crypto.KeyFormat) ([]byte, error) {
	switch format {
	case crypto.KeyFormatPEM:
		return formats.NewPEMExporter().Export(key, format)
	case crypto.KeyFormatJWK:
		return formats.NewJWKExporter().Export(key, format)
	default:
		return nil, crypto.ErrInvalidKeyFormat
	}
}
