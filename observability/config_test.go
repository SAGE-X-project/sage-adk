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

package observability

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Metrics.Enabled != true {
		t.Error("metrics should be enabled by default")
	}

	if config.Metrics.Port != 9090 {
		t.Errorf("default metrics port should be 9090, got %d", config.Metrics.Port)
	}

	if config.Logging.Level != "info" {
		t.Errorf("default log level should be 'info', got %s", config.Logging.Level)
	}

	if config.Health.Enabled != true {
		t.Error("health checks should be enabled by default")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid metrics port - negative",
			config: &Config{
				Metrics: MetricsConfig{Enabled: true, Port: -1, Path: "/metrics"},
			},
			wantErr: true,
			errMsg:  "metrics.port",
		},
		{
			name: "invalid metrics port - too high",
			config: &Config{
				Metrics: MetricsConfig{Enabled: true, Port: 70000, Path: "/metrics"},
			},
			wantErr: true,
			errMsg:  "metrics.port",
		},
		{
			name: "invalid metrics path",
			config: &Config{
				Metrics: MetricsConfig{Enabled: true, Port: 9090, Path: ""},
			},
			wantErr: true,
			errMsg:  "metrics.path",
		},
		{
			name: "invalid log level",
			config: &Config{
				Logging: LoggingConfig{Level: "invalid"},
			},
			wantErr: true,
			errMsg:  "logging.level",
		},
		{
			name: "invalid log format",
			config: &Config{
				Logging: LoggingConfig{Level: "info", Format: "invalid"},
			},
			wantErr: true,
			errMsg:  "logging.format",
		},
		{
			name: "invalid sampling rate - negative",
			config: &Config{
				Logging: LoggingConfig{Level: "info", SamplingRate: -0.1},
			},
			wantErr: true,
			errMsg:  "logging.sampling_rate",
		},
		{
			name: "invalid sampling rate - too high",
			config: &Config{
				Logging: LoggingConfig{Level: "info", SamplingRate: 1.5},
			},
			wantErr: true,
			errMsg:  "logging.sampling_rate",
		},
		{
			name: "tracing enabled but no endpoint",
			config: &Config{
				Tracing: TracingConfig{Enabled: true, Endpoint: ""},
			},
			wantErr: true,
			errMsg:  "tracing.endpoint",
		},
		{
			name: "invalid tracing sampling rate",
			config: &Config{
				Tracing: TracingConfig{
					Enabled:      true,
					Endpoint:     "http://jaeger:14268",
					SamplingRate: 2.0,
				},
			},
			wantErr: true,
			errMsg:  "tracing.sampling_rate",
		},
		{
			name: "invalid health port",
			config: &Config{
				Health: HealthConfig{Enabled: true, Port: 0},
			},
			wantErr: true,
			errMsg:  "health.port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}

				if tt.errMsg != "" {
					configErr, ok := err.(*ConfigError)
					if !ok {
						t.Errorf("expected ConfigError, got %T", err)
						return
					}

					if configErr.Field != tt.errMsg {
						t.Errorf("expected error field %s, got %s", tt.errMsg, configErr.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "test.field",
		Message: "test message",
	}

	expected := "observability config error: test.field: test message"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}
