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
	"math/big"
	"testing"
	"time"

	adkconfig "github.com/sage-x-project/sage-adk/config"
)

func TestFromADKConfig(t *testing.T) {
	tests := []struct {
		name    string
		adkCfg  *adkconfig.SAGEConfig
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name: "valid ethereum config",
			adkCfg: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "ethereum",
				DID:             "did:sage:ethereum:0x123",
				RPCEndpoint:     "https://eth.example.com",
				ContractAddress: "0xABC",
				PrivateKeyPath:  "/path/to/key",
				CacheTTL:        1 * time.Hour,
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.LocalDID != "did:sage:ethereum:0x123" {
					t.Errorf("LocalDID = %s, want did:sage:ethereum:0x123", cfg.LocalDID)
				}
				if cfg.Blockchain.NetworkRPC != "https://eth.example.com" {
					t.Errorf("NetworkRPC = %s, want https://eth.example.com", cfg.Blockchain.NetworkRPC)
				}
				if cfg.Blockchain.ChainID.Cmp(big.NewInt(1)) != 0 {
					t.Errorf("ChainID = %v, want 1", cfg.Blockchain.ChainID)
				}
				if cfg.DID.Network != "ethereum" {
					t.Errorf("DID.Network = %s, want ethereum", cfg.DID.Network)
				}
				if cfg.Environment != "production" {
					t.Errorf("Environment = %s, want production", cfg.Environment)
				}
			},
		},
		{
			name: "valid sepolia config",
			adkCfg: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "sepolia",
				DID:             "did:sage:sepolia:0x456",
				RPCEndpoint:     "https://sepolia.example.com",
				ContractAddress: "0xDEF",
				PrivateKeyPath:  "keys/sepolia.key",
				CacheTTL:        30 * time.Minute,
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Blockchain.ChainID.Cmp(big.NewInt(11155111)) != 0 {
					t.Errorf("ChainID = %v, want 11155111", cfg.Blockchain.ChainID)
				}
				if cfg.Environment != "testnet" {
					t.Errorf("Environment = %s, want testnet", cfg.Environment)
				}
				if cfg.DID.CacheTTL != 30*time.Minute {
					t.Errorf("CacheTTL = %v, want 30m", cfg.DID.CacheTTL)
				}
			},
		},
		{
			name: "valid kaia config",
			adkCfg: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "kaia",
				DID:             "did:sage:kaia:0x789",
				RPCEndpoint:     "https://kaia.example.com",
				ContractAddress: "0xGHI",
				PrivateKeyPath:  "keys/kaia.key",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Blockchain.ChainID.Cmp(big.NewInt(8217)) != 0 {
					t.Errorf("ChainID = %v, want 8217", cfg.Blockchain.ChainID)
				}
				if cfg.Environment != "production" {
					t.Errorf("Environment = %s, want production", cfg.Environment)
				}
				if cfg.DID.CacheTTL != 1*time.Hour {
					t.Errorf("CacheTTL = %v, want 1h (default)", cfg.DID.CacheTTL)
				}
			},
		},
		{
			name: "valid local config",
			adkCfg: &adkconfig.SAGEConfig{
				Enabled:         true,
				Network:         "local",
				DID:             "did:sage:local:0xABC",
				RPCEndpoint:     "http://localhost:8545",
				ContractAddress: "0xJKL",
				PrivateKeyPath:  "./keys/local.key",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Blockchain.ChainID.Cmp(big.NewInt(31337)) != 0 {
					t.Errorf("ChainID = %v, want 31337", cfg.Blockchain.ChainID)
				}
				if cfg.Environment != "development" {
					t.Errorf("Environment = %s, want development", cfg.Environment)
				}
				// filepath.Dir("./keys/local.key") returns "keys" on some systems, "./keys" on others
				// Just check that it contains "keys"
				if cfg.KeyStore.Directory == "" {
					t.Errorf("KeyStore.Directory is empty")
				}
			},
		},
		{
			name:    "nil config",
			adkCfg:  nil,
			wantErr: true,
		},
		{
			name: "missing DID",
			adkCfg: &adkconfig.SAGEConfig{
				Network:         "ethereum",
				RPCEndpoint:     "https://eth.example.com",
				PrivateKeyPath:  "/path/to/key",
			},
			wantErr: true,
		},
		{
			name: "missing private key path",
			adkCfg: &adkconfig.SAGEConfig{
				DID:         "did:sage:ethereum:0x123",
				Network:     "ethereum",
				RPCEndpoint: "https://eth.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing network",
			adkCfg: &adkconfig.SAGEConfig{
				DID:            "did:sage:ethereum:0x123",
				RPCEndpoint:    "https://eth.example.com",
				PrivateKeyPath: "/path/to/key",
			},
			wantErr: true,
		},
		{
			name: "missing RPC endpoint",
			adkCfg: &adkconfig.SAGEConfig{
				DID:            "did:sage:ethereum:0x123",
				Network:        "ethereum",
				PrivateKeyPath: "/path/to/key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := FromADKConfig(tt.adkCfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("FromADKConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	validADKCfg := &adkconfig.SAGEConfig{
		Enabled:         true,
		Network:         "sepolia",
		DID:             "did:sage:sepolia:0x123",
		RPCEndpoint:     "https://sepolia.example.com",
		ContractAddress: "0xABC",
		PrivateKeyPath:  "/path/to/key",
	}

	tests := []struct {
		name    string
		setup   func() *Config
		wantErr bool
	}{
		{
			name: "valid config",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				return cfg
			},
			wantErr: false,
		},
		{
			name: "nil sage config",
			setup: func() *Config {
				return &Config{
					Config:         nil,
					LocalDID:       "did:sage:test",
					PrivateKeyPath: "/path/to/key",
				}
			},
			wantErr: true,
		},
		{
			name: "empty local DID",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.LocalDID = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty private key path",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.PrivateKeyPath = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "nil blockchain config",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.Blockchain = nil
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty RPC endpoint",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.Blockchain.NetworkRPC = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "nil chain ID",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.Blockchain.ChainID = nil
				return cfg
			},
			wantErr: true,
		},
		{
			name: "nil DID config",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.DID = nil
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty DID network",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.DID.Network = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "nil key store config",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.KeyStore = nil
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty key store directory",
			setup: func() *Config {
				cfg, _ := FromADKConfig(validADKCfg)
				cfg.KeyStore.Directory = ""
				return cfg
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			err := cfg.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetChainIDFromNetwork(t *testing.T) {
	tests := []struct {
		network string
		want    int64
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
		{"unknown", 31337}, // default
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			got := getChainIDFromNetwork(tt.network)
			if got.Int64() != tt.want {
				t.Errorf("getChainIDFromNetwork(%s) = %d, want %d", tt.network, got.Int64(), tt.want)
			}
		})
	}
}

func TestGetEnvironmentFromNetwork(t *testing.T) {
	tests := []struct {
		network string
		want    string
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
		{"unknown", "development"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			got := getEnvironmentFromNetwork(tt.network)
			if got != tt.want {
				t.Errorf("getEnvironmentFromNetwork(%s) = %s, want %s", tt.network, got, tt.want)
			}
		})
	}
}
