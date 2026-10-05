// SPDX-License-Identifier: LGPL-3.0-or-later

// Package configmodel contains ADK-owned options for its legacy SAGE adapter.
// These data types are not the SAGE application's internal configuration API.
package configmodel

import (
	"math/big"
	"time"
)

// Config groups the connection options used by the ADK adapter.
type Config struct {
	Environment string            `json:"environment"`
	Blockchain  *BlockchainConfig `json:"blockchain"`
	DID         *DIDConfig        `json:"did"`
	KeyStore    *KeyStoreConfig   `json:"keystore"`
	Logging     *LoggingConfig    `json:"logging"`
}

// BlockchainConfig describes the adapter's registry connection.
type BlockchainConfig struct {
	NetworkRPC     string        `json:"network_rpc"`
	ContractAddr   string        `json:"contract_address"`
	ChainID        *big.Int      `json:"chain_id"`
	GasLimit       uint64        `json:"gas_limit"`
	MaxGasPrice    *big.Int      `json:"max_gas_price"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// DIDConfig describes the registry and local metadata cache.
type DIDConfig struct {
	RegistryAddress string        `json:"registry_address"`
	Method          string        `json:"method"`
	Network         string        `json:"network"`
	CacheSize       int           `json:"cache_size"`
	CacheTTL        time.Duration `json:"cache_ttl"`
}

// KeyStoreConfig describes local key storage.
type KeyStoreConfig struct {
	Type          string `json:"type"`
	Directory     string `json:"directory"`
	PassphraseEnv string `json:"passphrase_env"`
}

// LoggingConfig describes adapter logging preferences.
type LoggingConfig struct {
	Level    string `json:"level"`
	Format   string `json:"format"`
	Output   string `json:"output"`
	FilePath string `json:"file_path"`
}
