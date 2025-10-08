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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a file (YAML or JSON).
// The file format is determined by the file extension (.yaml, .yml, or .json).
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .yaml, .yml, or .json)", ext)
	}

	// Apply environment variable overrides
	if err := cfg.LoadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Validate configuration (includes SAGE-specific validation)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Additional SAGE-specific validation
	if err := cfg.ValidateSAGE(); err != nil {
		return nil, fmt.Errorf("invalid SAGE configuration: %w", err)
	}

	return cfg, nil
}

// LoadEnv loads configuration from environment variables.
// Environment variables take precedence over file-based configuration.
// Format: SAGE_ADK_<SECTION>_<FIELD> (e.g., SAGE_ADK_SAGE_DID)
func (c *Config) LoadEnv() error {
	// Agent config
	if v := os.Getenv("SAGE_ADK_AGENT_ID"); v != "" {
		c.Agent.ID = v
	}
	if v := os.Getenv("SAGE_ADK_AGENT_NAME"); v != "" {
		c.Agent.Name = v
	}

	// Server config
	if v := os.Getenv("SAGE_ADK_SERVER_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("SAGE_ADK_SERVER_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			c.Server.Port = port
		}
	}

	// SAGE config
	if v := os.Getenv("SAGE_ADK_SAGE_DID"); v != "" {
		c.SAGE.DID = v
	}
	if v := os.Getenv("SAGE_ADK_SAGE_PRIVATE_KEY_PATH"); v != "" {
		c.SAGE.PrivateKeyPath = v
	}
	if v := os.Getenv("SAGE_ADK_SAGE_NETWORK"); v != "" {
		c.SAGE.Network = v
	}
	if v := os.Getenv("SAGE_ADK_SAGE_RPC_ENDPOINT"); v != "" {
		c.SAGE.RPCEndpoint = v
	}
	if v := os.Getenv("SAGE_ADK_SAGE_CONTRACT_ADDRESS"); v != "" {
		c.SAGE.ContractAddress = v
	}
	if v := os.Getenv("SAGE_ADK_SAGE_ENABLED"); v != "" {
		c.SAGE.Enabled = v == "true" || v == "1"
	}

	// LLM config
	if v := os.Getenv("SAGE_ADK_LLM_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("SAGE_ADK_LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("SAGE_ADK_LLM_MODEL"); v != "" {
		c.LLM.Model = v
	}

	// Alternative shorter environment variable names (for convenience)
	if v := os.Getenv("SAGE_DID"); v != "" && c.SAGE.DID == "" {
		c.SAGE.DID = v
	}
	if v := os.Getenv("SAGE_PRIVATE_KEY_PATH"); v != "" && c.SAGE.PrivateKeyPath == "" {
		c.SAGE.PrivateKeyPath = v
	}
	if v := os.Getenv("SAGE_NETWORK"); v != "" && c.SAGE.Network == "" {
		c.SAGE.Network = v
	}
	if v := os.Getenv("SAGE_RPC_ENDPOINT"); v != "" && c.SAGE.RPCEndpoint == "" {
		c.SAGE.RPCEndpoint = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" && c.LLM.APIKey == "" {
		c.LLM.APIKey = v
	}

	return nil
}

// ValidateSAGE validates SAGE-specific configuration.
// This is called automatically by LoadFromFile.
func (c *Config) ValidateSAGE() error {
	// SAGE validation (if enabled)
	if c.SAGE.Enabled {
		if c.SAGE.DID == "" {
			return fmt.Errorf("SAGE.DID is required when SAGE is enabled")
		}
		if c.SAGE.PrivateKeyPath == "" {
			return fmt.Errorf("SAGE.PrivateKeyPath is required when SAGE is enabled")
		}
		if c.SAGE.Network == "" {
			return fmt.Errorf("SAGE.Network is required when SAGE is enabled")
		}
		if c.SAGE.RPCEndpoint == "" {
			return fmt.Errorf("SAGE.RPCEndpoint is required when SAGE is enabled")
		}
	}

	return nil
}
