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
	"fmt"
)

// Validate validates the entire configuration.
func (c *Config) Validate() error {
	if err := c.validateAgent(); err != nil {
		return err
	}

	if err := c.validateServer(); err != nil {
		return err
	}

	if err := c.validateProtocol(); err != nil {
		return err
	}

	if err := c.validateLLM(); err != nil {
		return err
	}

	if err := c.validateStorage(); err != nil {
		return err
	}

	return nil
}

// validateAgent validates agent configuration.
func (c *Config) validateAgent() error {
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name must not be empty")
	}

	return nil
}

// validateServer validates server configuration.
func (c *Config) validateServer() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive")
	}

	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive")
	}

	return nil
}

// validateProtocol validates protocol configuration.
func (c *Config) validateProtocol() error {
	validModes := map[string]bool{
		"a2a":  true,
		"sage": true,
		"auto": true,
	}

	if !validModes[c.Protocol.Mode] {
		return fmt.Errorf("protocol mode must be one of: a2a, sage, auto")
	}

	validDefaultProtocols := map[string]bool{
		"a2a":  true,
		"sage": true,
	}

	if !validDefaultProtocols[c.Protocol.DefaultProtocol] {
		return fmt.Errorf("default protocol must be one of: a2a, sage")
	}

	return nil
}

// validateLLM validates LLM configuration.
func (c *Config) validateLLM() error {
	// If provider is empty, skip validation (LLM is optional)
	if c.LLM.Provider == "" {
		return nil
	}

	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"gemini":    true,
	}

	if !validProviders[c.LLM.Provider] {
		return fmt.Errorf("LLM provider must be one of: openai, anthropic, gemini")
	}

	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key must not be empty")
	}

	if c.LLM.MaxTokens < 0 {
		return fmt.Errorf("LLM max tokens must not be negative")
	}

	return nil
}

// validateStorage validates storage configuration.
func (c *Config) validateStorage() error {
	validTypes := map[string]bool{
		"memory":   true,
		"redis":    true,
		"postgres": true,
	}

	if !validTypes[c.Storage.Type] {
		return fmt.Errorf("storage type must be one of: memory, redis, postgres")
	}

	if c.Storage.Type == "redis" {
		if err := c.validateRedis(); err != nil {
			return err
		}
	}

	if c.Storage.Type == "postgres" {
		if err := c.validatePostgres(); err != nil {
			return err
		}
	}

	return nil
}

// validateRedis validates Redis configuration.
func (c *Config) validateRedis() error {
	if c.Storage.Redis.Host == "" {
		return fmt.Errorf("redis host must not be empty")
	}

	if c.Storage.Redis.Port < 1 || c.Storage.Redis.Port > 65535 {
		return fmt.Errorf("redis port must be between 1 and 65535")
	}

	return nil
}

// validatePostgres validates PostgreSQL configuration.
func (c *Config) validatePostgres() error {
	if c.Storage.Postgres.Host == "" {
		return fmt.Errorf("postgres host must not be empty")
	}

	if c.Storage.Postgres.Port < 1 || c.Storage.Postgres.Port > 65535 {
		return fmt.Errorf("postgres port must be between 1 and 65535")
	}

	if c.Storage.Postgres.User == "" {
		return fmt.Errorf("postgres user must not be empty")
	}

	if c.Storage.Postgres.Database == "" {
		return fmt.Errorf("postgres database must not be empty")
	}

	return nil
}
