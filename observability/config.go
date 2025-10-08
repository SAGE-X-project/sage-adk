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

// Config holds all observability configuration.
type Config struct {
	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Tracing configuration
	Tracing TracingConfig `yaml:"tracing" json:"tracing"`

	// Health check configuration
	Health HealthConfig `yaml:"health" json:"health"`
}

// MetricsConfig configures metrics collection.
type MetricsConfig struct {
	// Enabled enables metrics collection
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Port for metrics endpoint
	Port int `yaml:"port" json:"port"`

	// Path for metrics endpoint
	Path string `yaml:"path" json:"path"`

	// Interval for metric collection in seconds
	Interval int `yaml:"interval" json:"interval"`
}

// LoggingConfig configures structured logging.
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error, fatal)
	Level string `yaml:"level" json:"level"`

	// Format is the log format (json, text)
	Format string `yaml:"format" json:"format"`

	// Output is the log destination (stdout, stderr, file)
	Output string `yaml:"output" json:"output"`

	// FilePath is the log file path (when Output=file)
	FilePath string `yaml:"file_path" json:"file_path"`

	// SamplingRate is the sampling rate for debug logs (0.0-1.0)
	SamplingRate float64 `yaml:"sampling_rate" json:"sampling_rate"`
}

// TracingConfig configures distributed tracing.
type TracingConfig struct {
	// Enabled enables distributed tracing
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Endpoint is the OTLP endpoint
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// ServiceName is the service name for traces
	ServiceName string `yaml:"service_name" json:"service_name"`

	// SamplingRate is the trace sampling rate (0.0-1.0)
	SamplingRate float64 `yaml:"sampling_rate" json:"sampling_rate"`
}

// HealthConfig configures health check endpoints.
type HealthConfig struct {
	// Enabled enables health check endpoints
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Port for health check endpoints
	Port int `yaml:"port" json:"port"`

	// LivenessPath is the liveness probe path
	LivenessPath string `yaml:"liveness_path" json:"liveness_path"`

	// ReadinessPath is the readiness probe path
	ReadinessPath string `yaml:"readiness_path" json:"readiness_path"`

	// StartupPath is the startup probe path
	StartupPath string `yaml:"startup_path" json:"startup_path"`
}

// DefaultConfig returns a default observability configuration.
func DefaultConfig() *Config {
	return &Config{
		Metrics: MetricsConfig{
			Enabled:  true,
			Port:     9090,
			Path:     "/metrics",
			Interval: 15,
		},
		Logging: LoggingConfig{
			Level:        "info",
			Format:       "json",
			Output:       "stdout",
			SamplingRate: 0.1,
		},
		Tracing: TracingConfig{
			Enabled:      false,
			ServiceName:  "sage-agent",
			SamplingRate: 0.1,
		},
		Health: HealthConfig{
			Enabled:       true,
			Port:          8080,
			LivenessPath:  "/health/live",
			ReadinessPath: "/health/ready",
			StartupPath:   "/health/startup",
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Metrics.Enabled {
		if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
			return &ConfigError{Field: "metrics.port", Message: "must be between 1 and 65535"}
		}
		if c.Metrics.Path == "" {
			return &ConfigError{Field: "metrics.path", Message: "must not be empty"}
		}
	}

	if c.Logging.Level != "" {
		validLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
		}
		if !validLevels[c.Logging.Level] {
			return &ConfigError{Field: "logging.level", Message: "must be one of: debug, info, warn, error, fatal"}
		}
	}

	if c.Logging.Format != "" && c.Logging.Format != "json" && c.Logging.Format != "text" {
		return &ConfigError{Field: "logging.format", Message: "must be 'json' or 'text'"}
	}

	if c.Logging.SamplingRate < 0.0 || c.Logging.SamplingRate > 1.0 {
		return &ConfigError{Field: "logging.sampling_rate", Message: "must be between 0.0 and 1.0"}
	}

	if c.Tracing.Enabled {
		if c.Tracing.Endpoint == "" {
			return &ConfigError{Field: "tracing.endpoint", Message: "must not be empty when tracing is enabled"}
		}
		if c.Tracing.SamplingRate < 0.0 || c.Tracing.SamplingRate > 1.0 {
			return &ConfigError{Field: "tracing.sampling_rate", Message: "must be between 0.0 and 1.0"}
		}
	}

	if c.Health.Enabled {
		if c.Health.Port <= 0 || c.Health.Port > 65535 {
			return &ConfigError{Field: "health.port", Message: "must be between 1 and 65535"}
		}
	}

	return nil
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "observability config error: " + e.Field + ": " + e.Message
}
