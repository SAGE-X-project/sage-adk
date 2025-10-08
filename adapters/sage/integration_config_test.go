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
	"path/filepath"
	"testing"
	"time"

	adkconfig "github.com/sage-x-project/sage-adk/config"
)

// TestIntegration_ConfigToTransport tests the full integration from ADK config to TransportManager.
func TestIntegration_ConfigToTransport(t *testing.T) {
	// Setup: Create temporary directory for test keys
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "integration_key.pem")

	// Step 1: Create KeyManager and generate a key
	km := NewKeyManager()
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Step 2: Save key to file
	if err := km.SaveToFile(keyPair, keyPath); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Step 3: Create ADK config
	adkCfg := &adkconfig.SAGEConfig{
		Enabled:         true,
		Network:         "sepolia",
		DID:             "did:sage:sepolia:0xIntegrationTest",
		RPCEndpoint:     "https://eth-sepolia.example.com",
		ContractAddress: "0xIntegrationTestContract",
		PrivateKeyPath:  keyPath,
		CacheTTL:        1 * time.Hour,
	}

	// Step 4: Convert to SAGE config
	cfg, err := FromADKConfig(adkCfg)
	if err != nil {
		t.Fatalf("FromADKConfig() failed: %v", err)
	}

	// Step 5: Validate config
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Step 6: Create TransportManager from config
	tm, err := NewTransportManagerFromConfig(cfg, km)
	if err != nil {
		t.Fatalf("NewTransportManagerFromConfig() failed: %v", err)
	}

	// Verify TransportManager is properly configured
	if tm == nil {
		t.Fatal("TransportManager is nil")
	}

	if tm.localDID != cfg.LocalDID {
		t.Errorf("TransportManager DID = %s, want %s", tm.localDID, cfg.LocalDID)
	}

	if tm.privateKey == nil {
		t.Error("TransportManager private key is nil")
	}

	if tm.publicKey == nil {
		t.Error("TransportManager public key is nil")
	}

	// Verify key consistency
	loadedKeyPair, err := km.LoadFromFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to reload key: %v", err)
	}

	extractedPrivKey, err := km.ExtractEd25519PrivateKey(loadedKeyPair)
	if err != nil {
		t.Fatalf("Failed to extract private key: %v", err)
	}

	if string(extractedPrivKey) != string(tm.privateKey) {
		t.Error("TransportManager private key does not match loaded key")
	}

	extractedPubKey, err := km.ExtractEd25519PublicKey(loadedKeyPair)
	if err != nil {
		t.Fatalf("Failed to extract public key: %v", err)
	}

	if string(extractedPubKey) != string(tm.publicKey) {
		t.Error("TransportManager public key does not match loaded key")
	}
}

// TestIntegration_KeyManagerRoundTrip tests key generation, save, and load cycle.
func TestIntegration_KeyManagerRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	km := NewKeyManager()

	// Test Ed25519 key round-trip
	t.Run("Ed25519 PEM", func(t *testing.T) {
		keyPath := filepath.Join(tmpDir, "ed25519.pem")

		// Generate
		original, err := km.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}

		// Save
		if err := km.SaveToFile(original, keyPath); err != nil {
			t.Fatalf("SaveToFile() failed: %v", err)
		}

		// Load
		loaded, err := km.LoadFromFile(keyPath)
		if err != nil {
			t.Fatalf("LoadFromFile() failed: %v", err)
		}

		// Extract and compare
		origPriv, _ := km.ExtractEd25519PrivateKey(original)
		loadPriv, _ := km.ExtractEd25519PrivateKey(loaded)

		if string(origPriv) != string(loadPriv) {
			t.Error("Private keys do not match after round-trip")
		}

		origPub, _ := km.ExtractEd25519PublicKey(original)
		loadPub, _ := km.ExtractEd25519PublicKey(loaded)

		if string(origPub) != string(loadPub) {
			t.Error("Public keys do not match after round-trip")
		}
	})

	// Test JWK format
	t.Run("Ed25519 JWK", func(t *testing.T) {
		keyPath := filepath.Join(tmpDir, "ed25519.jwk")

		// Generate
		original, err := km.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}

		// Save as JWK
		if err := km.SaveToFileWithFormat(original, keyPath, "jwk"); err != nil {
			t.Fatalf("SaveToFileWithFormat() failed: %v", err)
		}

		// Load
		loaded, err := km.LoadFromFile(keyPath)
		if err != nil {
			t.Fatalf("LoadFromFile() failed: %v", err)
		}

		if loaded.Type() != original.Type() {
			t.Errorf("Key type = %s, want %s", loaded.Type(), original.Type())
		}

		// Extract and compare
		origPriv, _ := km.ExtractEd25519PrivateKey(original)
		loadPriv, _ := km.ExtractEd25519PrivateKey(loaded)

		if string(origPriv) != string(loadPriv) {
			t.Error("Private keys do not match after JWK round-trip")
		}
	})
}

// TestIntegration_ConfigValidation tests config validation with various scenarios.
func TestIntegration_ConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "validation_key.pem")

	// Create a valid key
	km := NewKeyManager()
	keyPair, _ := km.Generate()
	km.SaveToFile(keyPair, keyPath)

	tests := []struct {
		name    string
		config  *adkconfig.SAGEConfig
		wantErr bool
	}{
		{
			name: "valid ethereum config",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "ethereum",
				DID:             "did:sage:ethereum:0x123",
				RPCEndpoint:     "https://eth.example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			},
			wantErr: false,
		},
		{
			name: "valid sepolia config",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "sepolia",
				DID:             "did:sage:sepolia:0x456",
				RPCEndpoint:     "https://sepolia.example.com",
				ContractAddress: "0xDEF",
				PrivateKeyPath:  keyPath,
			},
			wantErr: false,
		},
		{
			name: "valid kaia config",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "kaia",
				DID:             "did:sage:kaia:0x789",
				RPCEndpoint:     "https://kaia.example.com",
				ContractAddress: "0xGHI",
				PrivateKeyPath:  keyPath,
			},
			wantErr: false,
		},
		{
			name: "missing DID",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "ethereum",
				DID:             "",
				RPCEndpoint:     "https://eth.example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			},
			wantErr: true,
		},
		{
			name: "missing network",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "",
				DID:             "did:sage:ethereum:0x123",
				RPCEndpoint:     "https://eth.example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			},
			wantErr: true,
		},
		{
			name: "missing RPC endpoint",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "ethereum",
				DID:             "did:sage:ethereum:0x123",
				RPCEndpoint:     "",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			},
			wantErr: true,
		},
		{
			name: "missing private key path",
			config: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "ethereum",
				DID:             "did:sage:ethereum:0x123",
				RPCEndpoint:     "https://eth.example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := FromADKConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("FromADKConfig() should have failed")
				}
				return
			}

			if err != nil {
				t.Fatalf("FromADKConfig() failed: %v", err)
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() failed: %v", err)
			}
		})
	}
}

// TestIntegration_NetworkChainIDMapping tests network to chain ID mapping.
func TestIntegration_NetworkChainIDMapping(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "network_key.pem")

	km := NewKeyManager()
	keyPair, _ := km.Generate()
	km.SaveToFile(keyPair, keyPath)

	tests := []struct {
		network       string
		expectedChainID int64
	}{
		{"ethereum", 1},
		{"mainnet", 1},
		{"sepolia", 11155111},
		{"goerli", 5},
		{"kaia", 8217},
		{"cypress", 8217},
		{"kairos", 1001},
		{"kaia-testnet", 1001},
		{"local", 31337},
		{"localhost", 31337},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			adkCfg := &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         tt.network,
				DID:             "did:sage:" + tt.network + ":0x123",
				RPCEndpoint:     "https://" + tt.network + ".example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			}

			cfg, err := FromADKConfig(adkCfg)
			if err != nil {
				t.Fatalf("FromADKConfig() failed: %v", err)
			}

			if cfg.Blockchain.ChainID.Int64() != tt.expectedChainID {
				t.Errorf("ChainID = %d, want %d", cfg.Blockchain.ChainID.Int64(), tt.expectedChainID)
			}
		})
	}
}

// TestIntegration_EnvironmentDetection tests environment detection from network.
func TestIntegration_EnvironmentDetection(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "env_key.pem")

	km := NewKeyManager()
	keyPair, _ := km.Generate()
	km.SaveToFile(keyPair, keyPath)

	tests := []struct {
		network     string
		expectedEnv string
	}{
		{"ethereum", "production"},
		{"mainnet", "production"},
		{"kaia", "production"},
		{"cypress", "production"},
		{"sepolia", "testnet"},
		{"goerli", "testnet"},
		{"kairos", "testnet"},
		{"kaia-testnet", "testnet"},
		{"local", "development"},
		{"localhost", "development"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			adkCfg := &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         tt.network,
				DID:             "did:sage:" + tt.network + ":0x123",
				RPCEndpoint:     "https://" + tt.network + ".example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  keyPath,
			}

			cfg, err := FromADKConfig(adkCfg)
			if err != nil {
				t.Fatalf("FromADKConfig() failed: %v", err)
			}

			if cfg.Environment != tt.expectedEnv {
				t.Errorf("Environment = %s, want %s", cfg.Environment, tt.expectedEnv)
			}
		})
	}
}
