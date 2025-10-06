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

func TestConfig_Validate_ServerTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		server  ServerConfig
		wantErr bool
	}{
		{
			name: "negative read timeout",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         8080,
				ReadTimeout:  -1 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero read timeout",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         8080,
				ReadTimeout:  0,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative write timeout",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         8080,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero write timeout",
			server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         8080,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 0,
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

func TestConfig_Validate_Redis(t *testing.T) {
	tests := []struct {
		name    string
		storage StorageConfig
		wantErr bool
	}{
		{
			name: "redis with invalid port",
			storage: StorageConfig{
				Type: "redis",
				Redis: RedisConfig{
					Host: "localhost",
					Port: 70000,
				},
			},
			wantErr: true,
		},
		{
			name: "redis with zero port",
			storage: StorageConfig{
				Type: "redis",
				Redis: RedisConfig{
					Host: "localhost",
					Port: 0,
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

func TestConfig_Validate_Postgres(t *testing.T) {
	tests := []struct {
		name    string
		storage StorageConfig
		wantErr bool
	}{
		{
			name: "valid postgres",
			storage: StorageConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "testuser",
					Database: "testdb",
					SSLMode:  "disable",
				},
			},
			wantErr: false,
		},
		{
			name: "postgres without host",
			storage: StorageConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Port:     5432,
					User:     "testuser",
					Database: "testdb",
				},
			},
			wantErr: true,
		},
		{
			name: "postgres with invalid port",
			storage: StorageConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Host:     "localhost",
					Port:     70000,
					User:     "testuser",
					Database: "testdb",
				},
			},
			wantErr: true,
		},
		{
			name: "postgres without user",
			storage: StorageConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
				},
			},
			wantErr: true,
		},
		{
			name: "postgres without database",
			storage: StorageConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Host: "localhost",
					Port: 5432,
					User: "testuser",
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
