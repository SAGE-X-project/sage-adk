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

// Package config provides configuration management for SAGE ADK.
//
// The configuration system supports multiple sources with the following precedence:
//   1. Programmatic configuration (explicit Set calls)
//   2. Environment variables (prefixed with ADK_)
//   3. Configuration file (YAML)
//   4. Default values
//
// # Configuration Structure
//
// The configuration is organized into sections:
//   - Agent: Agent identity and metadata
//   - Server: HTTP server settings
//   - Protocol: Protocol mode and selection
//   - A2A: A2A protocol settings
//   - SAGE: SAGE protocol and security settings
//   - LLM: LLM provider configuration
//   - Storage: Storage backend configuration
//   - Logging: Logging configuration
//   - Metrics: Metrics and monitoring
//
// # Usage
//
// Loading configuration:
//
//	cfg, err := config.Load("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Using configuration manager:
//
//	manager := config.NewManager()
//	if err := manager.Load("config.yaml"); err != nil {
//	    log.Fatal(err)
//	}
//	cfg := manager.Get()
//
// Environment variable override:
//
//	export ADK_AGENT_NAME="my-agent"
//	export ADK_SERVER_PORT=9090
//	export ADK_LLM_PROVIDER="openai"
//	export ADK_LLM_API_KEY="sk-..."
//
// # Validation
//
// All configuration is validated before use. Validation rules include:
//   - Agent name must not be empty
//   - Server port must be between 1 and 65535
//   - Protocol mode must be "a2a", "sage", or "auto"
//   - LLM provider must be "openai", "anthropic", or "gemini"
//   - Storage type must be "memory", "redis", or "postgres"
//
// See the Config.Validate() method for complete validation rules.
package config
