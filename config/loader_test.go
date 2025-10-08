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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromFile_YAML(t *testing.T) {
	// Create temporary YAML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
agent:
  id: "test-agent-id"
  name: "test-agent"
  version: "1.0.0"

server:
  host: "localhost"
  port: 8080

sage:
  enabled: false
  did: "did:sage:test:123"
  private_key_path: "/path/to/key"
  network: "ethereum"
  rpc_endpoint: "http://localhost:8545"
  contract_address: "0xABC123"

llm:
  provider: "openai"
  api_key: "sk-test-key"
  model: "gpt-4"
  max_tokens: 1000
  temperature: 0.8
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify loaded values
	if cfg.Agent.ID != "test-agent-id" {
		t.Errorf("Agent.ID = %s, want test-agent-id", cfg.Agent.ID)
	}
	if cfg.Agent.Name != "test-agent" {
		t.Errorf("Agent.Name = %s, want test-agent", cfg.Agent.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.SAGE.Enabled {
		t.Error("SAGE.Enabled = true, want false")
	}
	if cfg.SAGE.DID != "did:sage:test:123" {
		t.Errorf("SAGE.DID = %s, want did:sage:test:123", cfg.SAGE.DID)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("LLM.Provider = %s, want openai", cfg.LLM.Provider)
	}
}

func TestLoadFromFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "agent": {
    "id": "json-agent",
    "name": "test-json-agent"
  },
  "sage": {
    "enabled": false,
    "did": "did:sage:json:456",
    "private_key_path": "/json/key",
    "network": "kaia",
    "rpc_endpoint": "http://kaia:8545"
  },
  "llm": {
    "provider": "openai",
    "api_key": "sk-json-key"
  }
}`

	if err := os.WriteFile(configPath, []byte(jsonContent), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Agent.ID != "json-agent" {
		t.Errorf("Agent.ID = %s, want json-agent", cfg.Agent.ID)
	}
	if cfg.SAGE.Network != "kaia" {
		t.Errorf("SAGE.Network = %s, want kaia", cfg.SAGE.Network)
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadFromFile_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
agent:
  name: test
  invalid: [
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadFromFile_UnsupportedExtension(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.txt")

	if err := os.WriteFile(configPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("Expected error for unsupported file extension, got nil")
	}
}

func TestLoadEnv(t *testing.T) {
	// Set environment variables
	testEnv := map[string]string{
		"SAGE_ADK_AGENT_ID":                "env-agent-id",
		"SAGE_ADK_AGENT_NAME":              "env-agent",
		"SAGE_ADK_SERVER_HOST":             "env-host",
		"SAGE_ADK_SERVER_PORT":             "9090",
		"SAGE_ADK_SAGE_DID":                "did:sage:env:789",
		"SAGE_ADK_SAGE_PRIVATE_KEY_PATH":   "/env/key",
		"SAGE_ADK_SAGE_NETWORK":            "sepolia",
		"SAGE_ADK_SAGE_RPC_ENDPOINT":       "http://sepolia:8545",
		"SAGE_ADK_SAGE_CONTRACT_ADDRESS":   "0xENV123",
		"SAGE_ADK_SAGE_ENABLED":            "true",
		"SAGE_ADK_LLM_PROVIDER":            "anthropic",
		"SAGE_ADK_LLM_API_KEY":             "sk-env-key",
		"SAGE_ADK_LLM_MODEL":               "claude-3",
	}

	for k, v := range testEnv {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg := DefaultConfig()
	if err := cfg.LoadEnv(); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Agent.ID", cfg.Agent.ID, "env-agent-id"},
		{"Agent.Name", cfg.Agent.Name, "env-agent"},
		{"Server.Host", cfg.Server.Host, "env-host"},
		{"Server.Port", cfg.Server.Port, 9090},
		{"SAGE.DID", cfg.SAGE.DID, "did:sage:env:789"},
		{"SAGE.PrivateKeyPath", cfg.SAGE.PrivateKeyPath, "/env/key"},
		{"SAGE.Network", cfg.SAGE.Network, "sepolia"},
		{"SAGE.RPCEndpoint", cfg.SAGE.RPCEndpoint, "http://sepolia:8545"},
		{"SAGE.ContractAddress", cfg.SAGE.ContractAddress, "0xENV123"},
		{"SAGE.Enabled", cfg.SAGE.Enabled, true},
		{"LLM.Provider", cfg.LLM.Provider, "anthropic"},
		{"LLM.APIKey", cfg.LLM.APIKey, "sk-env-key"},
		{"LLM.Model", cfg.LLM.Model, "claude-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestLoadEnv_ShortNames(t *testing.T) {
	// Test shorter convenience environment variable names
	testEnv := map[string]string{
		"SAGE_DID":              "did:sage:short:111",
		"SAGE_PRIVATE_KEY_PATH": "/short/key",
		"SAGE_NETWORK":          "ethereum",
		"SAGE_RPC_ENDPOINT":     "http://short:8545",
		"OPENAI_API_KEY":        "sk-short-key",
	}

	for k, v := range testEnv {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg := DefaultConfig()
	if err := cfg.LoadEnv(); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	if cfg.SAGE.DID != "did:sage:short:111" {
		t.Errorf("SAGE.DID = %s, want did:sage:short:111", cfg.SAGE.DID)
	}
	if cfg.SAGE.PrivateKeyPath != "/short/key" {
		t.Errorf("SAGE.PrivateKeyPath = %s, want /short/key", cfg.SAGE.PrivateKeyPath)
	}
	if cfg.LLM.APIKey != "sk-short-key" {
		t.Errorf("LLM.APIKey = %s, want sk-short-key", cfg.LLM.APIKey)
	}
}

func TestLoadEnv_Precedence(t *testing.T) {
	// Long form should take precedence over short form
	os.Setenv("SAGE_DID", "did:sage:short")
	os.Setenv("SAGE_ADK_SAGE_DID", "did:sage:long")
	defer os.Unsetenv("SAGE_DID")
	defer os.Unsetenv("SAGE_ADK_SAGE_DID")

	cfg := DefaultConfig()
	if err := cfg.LoadEnv(); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	if cfg.SAGE.DID != "did:sage:long" {
		t.Errorf("SAGE.DID = %s, want did:sage:long (long form should take precedence)", cfg.SAGE.DID)
	}
}

func TestValidateSAGE(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid SAGE config",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled:         true,
					DID:             "did:sage:test:123",
					PrivateKeyPath:  "/path/to/key",
					Network:         "ethereum",
					RPCEndpoint:     "http://localhost:8545",
					ContractAddress: "0xABC",
				},
			},
			wantErr: false,
		},
		{
			name: "missing DID",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled:        true,
					PrivateKeyPath: "/path/to/key",
					Network:        "ethereum",
					RPCEndpoint:    "http://localhost:8545",
				},
			},
			wantErr: true,
		},
		{
			name: "missing PrivateKeyPath",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled:     true,
					DID:         "did:sage:test:123",
					Network:     "ethereum",
					RPCEndpoint: "http://localhost:8545",
				},
			},
			wantErr: true,
		},
		{
			name: "missing Network",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled:        true,
					DID:            "did:sage:test:123",
					PrivateKeyPath: "/path/to/key",
					RPCEndpoint:    "http://localhost:8545",
				},
			},
			wantErr: true,
		},
		{
			name: "missing RPCEndpoint",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled:        true,
					DID:            "did:sage:test:123",
					PrivateKeyPath: "/path/to/key",
					Network:        "ethereum",
				},
			},
			wantErr: true,
		},
		{
			name: "SAGE disabled - no validation",
			config: &Config{
				SAGE: SAGEConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateSAGE()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSAGE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_LLM(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid LLM config",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.LLM.Provider = "openai"
				cfg.LLM.APIKey = "sk-test"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "missing API key",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.LLM.Provider = "openai"
				cfg.LLM.APIKey = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.LLM.Provider = "invalid"
				cfg.LLM.APIKey = "sk-test"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "no provider - no validation",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.LLM.Provider = ""
				return cfg
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile_WithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
agent:
  name: "test-agent"
sage:
  enabled: true
  did: "did:sage:file:123"
  private_key_path: "/file/key"
  network: "ethereum"
  rpc_endpoint: "http://file:8545"

llm:
  provider: "openai"
  api_key: "sk-file-key"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables to override (but keep required fields from file)
	os.Setenv("SAGE_ADK_SAGE_DID", "did:sage:env:override")
	os.Setenv("SAGE_ADK_LLM_API_KEY", "sk-env-override")
	defer os.Unsetenv("SAGE_ADK_SAGE_DID")
	defer os.Unsetenv("SAGE_ADK_LLM_API_KEY")

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Environment variables should override file values
	if cfg.SAGE.DID != "did:sage:env:override" {
		t.Errorf("SAGE.DID = %s, want did:sage:env:override (env should override file)", cfg.SAGE.DID)
	}
	if cfg.LLM.APIKey != "sk-env-override" {
		t.Errorf("LLM.APIKey = %s, want sk-env-override (env should override file)", cfg.LLM.APIKey)
	}
	// File values should remain for non-overridden fields
	if cfg.SAGE.PrivateKeyPath != "/file/key" {
		t.Errorf("SAGE.PrivateKeyPath = %s, want /file/key (file value should be preserved)", cfg.SAGE.PrivateKeyPath)
	}
	if cfg.SAGE.Network != "ethereum" {
		t.Errorf("SAGE.Network = %s, want ethereum (file value should be preserved)", cfg.SAGE.Network)
	}
	if cfg.SAGE.RPCEndpoint != "http://file:8545" {
		t.Errorf("SAGE.RPCEndpoint = %s, want http://file:8545 (file value should be preserved)", cfg.SAGE.RPCEndpoint)
	}
}

func TestLoadFromFile_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with SAGE enabled but missing required fields
	yamlContent := `
agent:
  name: "test-agent"
sage:
  enabled: true
  did: "did:sage:test:123"
  # Missing private_key_path, network, rpc_endpoint
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("Expected validation error for incomplete SAGE config, got nil")
	}
}

func TestDefaultConfigPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config - most fields should use defaults
	yamlContent := `
agent:
  name: "minimal-agent"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Custom values should be set
	if cfg.Agent.Name != "minimal-agent" {
		t.Errorf("Agent.Name = %s, want minimal-agent", cfg.Agent.Name)
	}

	// Default values should be preserved
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080 (default)", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 30s (default)", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Type != "memory" {
		t.Errorf("Storage.Type = %s, want memory (default)", cfg.Storage.Type)
	}
}
