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
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	sageconfig "github.com/sage-x-project/sage/config"
	adkconfig "github.com/sage-x-project/sage-adk/config"
)

// Config wraps sage library config with ADK-specific extensions.
// It provides a bridge between sage-adk configuration and the sage library.
type Config struct {
	// Embed sage library config
	*sageconfig.Config

	// ADK-specific fields
	LocalDID       string // The local agent's DID
	PrivateKeyPath string // Path to the private key file
}

// FromADKConfig converts ADK SAGE configuration to sage library configuration.
// This function maps sage-adk's simplified configuration to the more detailed
// sage library configuration structure.
func FromADKConfig(adkCfg *adkconfig.SAGEConfig) (*Config, error) {
	if adkCfg == nil {
		return nil, fmt.Errorf("ADK config cannot be nil")
	}

	// Validate required fields
	if adkCfg.DID == "" {
		return nil, fmt.Errorf("DID is required")
	}
	if adkCfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("private key path is required")
	}
	if adkCfg.Network == "" {
		return nil, fmt.Errorf("network is required")
	}
	if adkCfg.RPCEndpoint == "" {
		return nil, fmt.Errorf("RPC endpoint is required")
	}

	// Map chain ID from network name
	chainID := getChainIDFromNetwork(adkCfg.Network)

	// Create blockchain configuration
	blockchainCfg := &sageconfig.BlockchainConfig{
		NetworkRPC:     adkCfg.RPCEndpoint,
		ContractAddr:   adkCfg.ContractAddress,
		ChainID:        chainID,
		GasLimit:       3000000,                    // Default gas limit
		MaxGasPrice:    big.NewInt(100000000000),   // 100 gwei
		MaxRetries:     3,                          // Default retries
		RetryDelay:     time.Second,                // 1 second retry delay
		RequestTimeout: 30 * time.Second,           // 30 second timeout
	}

	// Create DID configuration
	didCfg := &sageconfig.DIDConfig{
		RegistryAddress: adkCfg.ContractAddress,
		Method:          "sage",
		Network:         adkCfg.Network,
		CacheSize:       1000,    // Default cache size
		CacheTTL:        adkCfg.CacheTTL,
	}

	// Default cache TTL if not specified
	if didCfg.CacheTTL == 0 {
		didCfg.CacheTTL = 1 * time.Hour
	}

	// Create key store configuration
	keyStoreDir := filepath.Dir(adkCfg.PrivateKeyPath)
	// Keep the directory as-is, don't change "." to "keys"

	keyStoreCfg := &sageconfig.KeyStoreConfig{
		Type:      "file",
		Directory: keyStoreDir,
	}

	// Create logging configuration (optional, use defaults)
	loggingCfg := &sageconfig.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	// Create sage library config
	sageCfg := &sageconfig.Config{
		Environment: getEnvironmentFromNetwork(adkCfg.Network),
		Blockchain:  blockchainCfg,
		DID:         didCfg,
		KeyStore:    keyStoreCfg,
		Logging:     loggingCfg,
	}

	// Return wrapped config
	return &Config{
		Config:         sageCfg,
		LocalDID:       adkCfg.DID,
		PrivateKeyPath: adkCfg.PrivateKeyPath,
	}, nil
}

// getChainIDFromNetwork returns the chain ID for a given network name.
func getChainIDFromNetwork(network string) *big.Int {
	switch network {
	case "ethereum", "mainnet":
		return big.NewInt(1)
	case "sepolia":
		return big.NewInt(11155111)
	case "goerli":
		return big.NewInt(5)
	case "kaia", "cypress":
		return big.NewInt(8217)
	case "kairos", "kaia-testnet":
		return big.NewInt(1001)
	case "local", "localhost":
		return big.NewInt(31337)
	default:
		// Default to local network
		return big.NewInt(31337)
	}
}

// getEnvironmentFromNetwork determines the environment type from network name.
func getEnvironmentFromNetwork(network string) string {
	switch network {
	case "ethereum", "mainnet", "kaia", "cypress":
		return "production"
	case "sepolia", "goerli", "kairos", "kaia-testnet":
		return "testnet"
	case "local", "localhost":
		return "development"
	default:
		return "development"
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Config == nil {
		return fmt.Errorf("sage config cannot be nil")
	}

	if c.LocalDID == "" {
		return fmt.Errorf("local DID is required")
	}

	if c.PrivateKeyPath == "" {
		return fmt.Errorf("private key path is required")
	}

	// Validate blockchain config
	if c.Blockchain == nil {
		return fmt.Errorf("blockchain config is required")
	}
	if c.Blockchain.NetworkRPC == "" {
		return fmt.Errorf("blockchain RPC endpoint is required")
	}
	if c.Blockchain.ChainID == nil {
		return fmt.Errorf("blockchain chain ID is required")
	}

	// Validate DID config
	if c.DID == nil {
		return fmt.Errorf("DID config is required")
	}
	if c.DID.Network == "" {
		return fmt.Errorf("DID network is required")
	}

	// Validate key store config
	if c.KeyStore == nil {
		return fmt.Errorf("key store config is required")
	}
	if c.KeyStore.Directory == "" {
		return fmt.Errorf("key store directory is required")
	}

	return nil
}
