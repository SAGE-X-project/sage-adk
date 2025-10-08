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

package config

import (
	"time"
)

// Config represents the complete configuration for SAGE ADK.
type Config struct {
	Agent    AgentConfig
	Server   ServerConfig
	Protocol ProtocolConfig
	A2A      A2AConfig
	SAGE     SAGEConfig
	LLM      LLMConfig
	Storage  StorageConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
}

// AgentConfig contains agent identity and metadata.
type AgentConfig struct {
	ID          string
	Name        string
	Description string
	Version     string
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// ProtocolConfig contains protocol mode and selection settings.
type ProtocolConfig struct {
	Mode            string // "a2a", "sage", "auto"
	AutoDetect      bool
	DefaultProtocol string // "a2a", "sage"
}

// A2AConfig contains A2A protocol settings.
type A2AConfig struct {
	Enabled     bool
	ServerURL   string
	Version     string
	TaskTimeout time.Duration
	Timeout     int    // Timeout in seconds
	UserAgent   string // User-Agent header
}

// SAGEConfig contains SAGE protocol and security settings.
type SAGEConfig struct {
	Enabled         bool          `json:"enabled" yaml:"enabled"`
	Network         string        `json:"network" yaml:"network"`                   // "ethereum", "kaia", "sepolia", etc.
	DID             string        `json:"did" yaml:"did"`
	RPCEndpoint     string        `json:"rpc_endpoint" yaml:"rpc_endpoint"`         // RPC endpoint URL
	ContractAddress string        `json:"contract_address" yaml:"contract_address"` // Smart contract address
	PrivateKeyPath  string        `json:"private_key_path" yaml:"private_key_path"`
	CacheEnabled    bool          `json:"cache_enabled" yaml:"cache_enabled"`
	CacheTTL        time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
}

// LLMConfig contains LLM provider configuration.
type LLMConfig struct {
	Provider    string        `json:"provider" yaml:"provider"`       // "openai", "anthropic", "gemini"
	APIKey      string        `json:"api_key" yaml:"api_key"`
	Model       string        `json:"model" yaml:"model"`
	MaxTokens   int           `json:"max_tokens" yaml:"max_tokens"`
	Temperature float64       `json:"temperature" yaml:"temperature"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
}

// StorageConfig contains storage backend configuration.
type StorageConfig struct {
	Type     string // "memory", "redis", "postgres"
	Redis    RedisConfig
	Postgres PostgresConfig
}

// RedisConfig contains Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// PostgresConfig contains PostgreSQL connection settings.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level      string // "debug", "info", "warn", "error"
	Format     string // "json", "text"
	OutputPath string
}

// MetricsConfig contains metrics and monitoring configuration.
type MetricsConfig struct {
	Enabled bool
	Port    int
	Path    string
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:    "sage-agent",
			Version: "0.1.0",
		},
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Protocol: ProtocolConfig{
			Mode:            "auto",
			AutoDetect:      true,
			DefaultProtocol: "a2a",
		},
		A2A: A2AConfig{
			Enabled:     true,
			Version:     "1.0",
			TaskTimeout: 5 * time.Minute,
		},
		SAGE: SAGEConfig{
			Enabled:      false,
			Network:      "ethereum",
			CacheEnabled: true,
			CacheTTL:     1 * time.Hour,
		},
		LLM: LLMConfig{
			Provider:    "", // Provider must be set when LLM is used
			MaxTokens:   2000,
			Temperature: 0.7,
			Timeout:     30 * time.Second,
		},
		Storage: StorageConfig{
			Type: "memory",
			Redis: RedisConfig{
				Host: "localhost",
				Port: 6379,
				DB:   0,
			},
			Postgres: PostgresConfig{
				Host:    "localhost",
				Port:    5432,
				SSLMode: "disable",
			},
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled: false,
			Port:    9090,
			Path:    "/metrics",
		},
	}
}

// NewConfig creates a new default configuration.
// This is an alias for DefaultConfig().
func NewConfig() *Config {
	return DefaultConfig()
}
