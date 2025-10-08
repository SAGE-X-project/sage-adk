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
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage/crypto"
)

func TestKeyManager_Generate(t *testing.T) {
	km := NewKeyManager()

	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if keyPair == nil {
		t.Fatal("Generate() returned nil key pair")
	}

	if keyPair.Type() != crypto.KeyTypeEd25519 {
		t.Errorf("Generated key type = %s, want %s", keyPair.Type(), crypto.KeyTypeEd25519)
	}

	if keyPair.ID() == "" {
		t.Error("Generated key has empty ID")
	}
}

func TestKeyManager_GenerateWithType(t *testing.T) {
	km := NewKeyManager()

	tests := []struct {
		name    string
		keyType crypto.KeyType
		wantErr bool
	}{
		{"Ed25519", crypto.KeyTypeEd25519, false},
		{"Secp256k1", crypto.KeyTypeSecp256k1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPair, err := km.GenerateWithType(tt.keyType)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateWithType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if keyPair.Type() != tt.keyType {
					t.Errorf("Generated key type = %s, want %s", keyPair.Type(), tt.keyType)
				}
			}
		})
	}
}

func TestKeyManager_SaveAndLoad_PEM(t *testing.T) {
	km := NewKeyManager()

	// Generate a key pair
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Save to temporary file
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key.pem")

	if err := km.SaveToFile(keyPair, keyPath); err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	// Verify file exists and has correct permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Key file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Load the key pair back
	loadedKeyPair, err := km.LoadFromFile(keyPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// Verify loaded key matches original
	if loadedKeyPair.Type() != keyPair.Type() {
		t.Errorf("Loaded key type = %s, want %s", loadedKeyPair.Type(), keyPair.Type())
	}

	// Compare public keys
	if string(loadedKeyPair.PublicKey().(ed25519.PublicKey)) != string(keyPair.PublicKey().(ed25519.PublicKey)) {
		t.Error("Loaded public key does not match original")
	}
}

func TestKeyManager_SaveAndLoad_JWK(t *testing.T) {
	km := NewKeyManager()

	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key.jwk")

	// Save as JWK
	if err := km.SaveToFileWithFormat(keyPair, keyPath, crypto.KeyFormatJWK); err != nil {
		t.Fatalf("SaveToFileWithFormat() failed: %v", err)
	}

	// Load from JWK
	loadedKeyPair, err := km.LoadFromFile(keyPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if loadedKeyPair.Type() != keyPair.Type() {
		t.Errorf("Loaded key type = %s, want %s", loadedKeyPair.Type(), keyPair.Type())
	}
}

func TestKeyManager_LoadFromFile_NotFound(t *testing.T) {
	km := NewKeyManager()

	_, err := km.LoadFromFile("/nonexistent/path/key.pem")
	if err == nil {
		t.Error("LoadFromFile() should fail for nonexistent file")
	}
}

func TestKeyManager_LoadFromFile_InvalidFormat(t *testing.T) {
	km := NewKeyManager()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "invalid_key.txt")

	// Write invalid data
	if err := os.WriteFile(keyPath, []byte("invalid key data"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := km.LoadFromFile(keyPath)
	if err == nil {
		t.Error("LoadFromFile() should fail for invalid key format")
	}
}

func TestKeyManager_StoreAndLoad(t *testing.T) {
	km := NewKeyManager()

	// Generate and store a key pair
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if err := km.Store(keyPair); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Load the key pair by ID
	loadedKeyPair, err := km.Load(keyPair.ID())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedKeyPair.ID() != keyPair.ID() {
		t.Errorf("Loaded key ID = %s, want %s", loadedKeyPair.ID(), keyPair.ID())
	}
}

func TestKeyManager_List(t *testing.T) {
	km := NewKeyManager()

	// Initially should be empty
	ids, err := km.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	initialCount := len(ids)

	// Store a key pair
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if err := km.Store(keyPair); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// List should now include the new key
	ids, err = km.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(ids) != initialCount+1 {
		t.Errorf("List() returned %d keys, want %d", len(ids), initialCount+1)
	}

	// Verify our key is in the list
	found := false
	for _, id := range ids {
		if id == keyPair.ID() {
			found = true
			break
		}
	}

	if !found {
		t.Error("Stored key ID not found in list")
	}
}

func TestKeyManager_Delete(t *testing.T) {
	km := NewKeyManager()

	// Generate and store a key pair
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if err := km.Store(keyPair); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Delete the key pair
	if err := km.Delete(keyPair.ID()); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Load should fail
	_, err = km.Load(keyPair.ID())
	if err == nil {
		t.Error("Load() should fail after Delete()")
	}
}

func TestKeyManager_ExtractEd25519PrivateKey(t *testing.T) {
	km := NewKeyManager()

	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	privateKey, err := km.ExtractEd25519PrivateKey(keyPair)
	if err != nil {
		t.Fatalf("ExtractEd25519PrivateKey() failed: %v", err)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		t.Errorf("Private key size = %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
}

func TestKeyManager_ExtractEd25519PublicKey(t *testing.T) {
	km := NewKeyManager()

	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	publicKey, err := km.ExtractEd25519PublicKey(keyPair)
	if err != nil {
		t.Fatalf("ExtractEd25519PublicKey() failed: %v", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		t.Errorf("Public key size = %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}

	// Verify it matches the key pair's public key
	if string(publicKey) != string(keyPair.PublicKey().(ed25519.PublicKey)) {
		t.Error("Extracted public key does not match key pair's public key")
	}
}

func TestKeyManager_ExtractEd25519_WrongKeyType(t *testing.T) {
	km := NewKeyManager()

	// Generate Secp256k1 key
	keyPair, err := km.GenerateWithType(crypto.KeyTypeSecp256k1)
	if err != nil {
		t.Fatalf("GenerateWithType() failed: %v", err)
	}

	// Extracting Ed25519 from Secp256k1 should fail
	_, err = km.ExtractEd25519PrivateKey(keyPair)
	if err == nil {
		t.Error("ExtractEd25519PrivateKey() should fail for non-Ed25519 key")
	}

	_, err = km.ExtractEd25519PublicKey(keyPair)
	if err == nil {
		t.Error("ExtractEd25519PublicKey() should fail for non-Ed25519 key")
	}
}

func TestKeyManager_RoundTrip(t *testing.T) {
	km := NewKeyManager()
	tmpDir := t.TempDir()

	// Generate → Save → Load → Verify
	original, err := km.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "roundtrip.pem")

	if err := km.SaveToFile(original, keyPath); err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	loaded, err := km.LoadFromFile(keyPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// Extract and compare keys
	originalPriv, _ := km.ExtractEd25519PrivateKey(original)
	loadedPriv, _ := km.ExtractEd25519PrivateKey(loaded)

	if string(originalPriv) != string(loadedPriv) {
		t.Error("Round-trip: private keys do not match")
	}

	originalPub, _ := km.ExtractEd25519PublicKey(original)
	loadedPub, _ := km.ExtractEd25519PublicKey(loaded)

	if string(originalPub) != string(loadedPub) {
		t.Error("Round-trip: public keys do not match")
	}
}
