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
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() should not return nil")
	}

	// Test agent defaults
	if cfg.Agent.Name == "" {
		t.Error("Agent.Name should have default value")
	}

	// Test server defaults
	if cfg.Server.Port == 0 {
		t.Error("Server.Port should have default value")
	}

	if cfg.Server.ReadTimeout == 0 {
		t.Error("Server.ReadTimeout should have default value")
	}

	// Test protocol defaults
	if cfg.Protocol.Mode == "" {
		t.Error("Protocol.Mode should have default value")
	}
}

func TestConfig_Validate_Success(t *testing.T) {
	cfg := DefaultConfig()

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for default config", err)
	}
}

func TestConfig_Validate_Agent(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid agent",
			config: &Config{
				Agent: AgentConfig{
					Name:    "test-agent",
					Version: "1.0.0",
				},
				Server:   DefaultConfig().Server,
				Protocol: DefaultConfig().Protocol,
				Storage:  DefaultConfig().Storage,
			},
			wantErr: false,
		},
		{
			name: "empty agent name",
			config: &Config{
				Agent: AgentConfig{
					Name:    "",
					Version: "1.0.0",
				},
				Server:   DefaultConfig().Server,
				Protocol: DefaultConfig().Protocol,
				Storage:  DefaultConfig().Storage,
			},
			wantErr: true,
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

func TestConfig_Validate_Server(t *testing.T) {
	tests := []struct {
		name    string
		server  ServerConfig
		wantErr bool
	}{
		{
			name: "valid server",
			server: ServerConfig{
				Host:            "0.0.0.0",
				Port:            8080,
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				ShutdownTimeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "port too low",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         0,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         70000,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Server = tt.server

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Protocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol ProtocolConfig
		wantErr  bool
	}{
		{
			name: "valid protocol a2a",
			protocol: ProtocolConfig{
				Mode:            "a2a",
				DefaultProtocol: "a2a",
			},
			wantErr: false,
		},
		{
			name: "valid protocol sage",
			protocol: ProtocolConfig{
				Mode:            "sage",
				DefaultProtocol: "sage",
			},
			wantErr: false,
		},
		{
			name: "valid protocol auto",
			protocol: ProtocolConfig{
				Mode:            "auto",
				DefaultProtocol: "a2a",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			protocol: ProtocolConfig{
				Mode:            "invalid",
				DefaultProtocol: "a2a",
			},
			wantErr: true,
		},
		{
			name: "invalid default protocol",
			protocol: ProtocolConfig{
				Mode:            "auto",
				DefaultProtocol: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Protocol = tt.protocol

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_LLM(t *testing.T) {
	tests := []struct {
		name    string
		llm     LLMConfig
		wantErr bool
	}{
		{
			name: "valid LLM config",
			llm: LLMConfig{
				Provider:  "openai",
				APIKey:    "sk-test",
				Model:     "gpt-4",
				MaxTokens: 1000,
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			llm: LLMConfig{
				Provider:  "invalid",
				APIKey:    "key",
				MaxTokens: 1000,
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			llm: LLMConfig{
				Provider:  "openai",
				APIKey:    "",
				MaxTokens: 1000,
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			llm: LLMConfig{
				Provider:  "openai",
				APIKey:    "key",
				MaxTokens: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.LLM = tt.llm

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Storage(t *testing.T) {
	tests := []struct {
		name    string
		storage StorageConfig
		wantErr bool
	}{
		{
			name: "valid memory storage",
			storage: StorageConfig{
				Type: "memory",
			},
			wantErr: false,
		},
		{
			name: "valid redis storage",
			storage: StorageConfig{
				Type: "redis",
				Redis: RedisConfig{
					Host: "localhost",
					Port: 6379,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid storage type",
			storage: StorageConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "redis without host",
			storage: StorageConfig{
				Type: "redis",
				Redis: RedisConfig{
					Port: 6379,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage = tt.storage

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
