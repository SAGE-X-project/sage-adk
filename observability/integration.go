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
	"context"
	"net/http"

	"github.com/sage-x-project/sage-adk/observability/health"
	"github.com/sage-x-project/sage-adk/observability/logging"
	"github.com/sage-x-project/sage-adk/observability/metrics"
)

// Manager manages all observability components for an agent.
type Manager struct {
	logger          logging.Logger
	collector       metrics.Collector
	agentMetrics    *metrics.AgentMetrics
	llmMetrics      *metrics.LLMMetrics
	middleware      *Middleware
	livenessChecker *health.LivenessChecker
	startupChecker  *health.StartupChecker
	readinessChecker *health.ReadinessChecker
}

// ManagerConfig configures the observability manager.
type ManagerConfig struct {
	// AgentID is the unique identifier for the agent
	AgentID string

	// Config is the observability configuration
	Config *Config
}

// NewManager creates a new observability manager.
//
// Example:
//
//	manager, err := observability.NewManager(&observability.ManagerConfig{
//	    AgentID: "my-agent",
//	    Config:  &observability.Config{...},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Shutdown(context.Background())
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	// Validate config
	if err := cfg.Config.Validate(); err != nil {
		return nil, err
	}

	// Create logger
	logger := logging.NewStructuredLogger(logging.Level(cfg.Config.Logging.Level))
	logger.SetSamplingRate(cfg.Config.Logging.SamplingRate)

	// Create metrics collector
	collector := metrics.NewPrometheusCollector()

	// Create agent and LLM metrics
	agentMetrics := metrics.NewAgentMetrics(collector)
	llmMetrics := metrics.NewLLMMetrics(collector)

	// Create middleware
	middleware := NewMiddleware(logger, agentMetrics, cfg.AgentID)

	// Create health checkers
	livenessChecker := health.NewLivenessChecker()
	startupChecker := health.NewStartupChecker()
	readinessChecker := health.NewReadinessChecker(startupChecker)

	// Mark agent as running
	livenessChecker.MarkRunning()

	return &Manager{
		logger:           logger,
		collector:        collector,
		agentMetrics:     agentMetrics,
		llmMetrics:       llmMetrics,
		middleware:       middleware,
		livenessChecker:  livenessChecker,
		startupChecker:   startupChecker,
		readinessChecker: readinessChecker,
	}, nil
}

// Logger returns the logger.
func (m *Manager) Logger() logging.Logger {
	return m.logger
}

// Collector returns the metrics collector.
func (m *Manager) Collector() metrics.Collector {
	return m.collector
}

// AgentMetrics returns the agent metrics.
func (m *Manager) AgentMetrics() *metrics.AgentMetrics {
	return m.agentMetrics
}

// LLMMetrics returns the LLM metrics.
func (m *Manager) LLMMetrics() *metrics.LLMMetrics {
	return m.llmMetrics
}

// Middleware returns the HTTP middleware.
func (m *Manager) Middleware() *Middleware {
	return m.middleware
}

// LivenessChecker returns the liveness checker.
func (m *Manager) LivenessChecker() *health.LivenessChecker {
	return m.livenessChecker
}

// StartupChecker returns the startup checker.
func (m *Manager) StartupChecker() *health.StartupChecker {
	return m.startupChecker
}

// ReadinessChecker returns the readiness checker.
func (m *Manager) ReadinessChecker() *health.ReadinessChecker {
	return m.readinessChecker
}

// MarkReady marks the agent as ready to serve traffic.
func (m *Manager) MarkReady() {
	m.startupChecker.MarkReady()
}

// AddReadinessCheck adds a health check to the readiness checker.
func (m *Manager) AddReadinessCheck(checker health.Checker) {
	m.readinessChecker.AddCheck(checker)
}

// HTTPHandler returns an http.Handler for exposing observability endpoints.
//
// It mounts the following endpoints:
//   - /metrics - Prometheus metrics
//   - /health/live - Liveness probe
//   - /health/ready - Readiness probe
//   - /health/startup - Startup probe
func (m *Manager) HTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle("/metrics", m.collector.Handler())

	// Health check endpoints
	mux.Handle("/health/live", health.Handler(m.livenessChecker))
	mux.Handle("/health/ready", health.Handler(m.readinessChecker))
	mux.Handle("/health/startup", health.Handler(m.startupChecker))

	return mux
}

// Shutdown gracefully shuts down the observability manager.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info(ctx, "shutting down observability manager")
	m.livenessChecker.MarkStopped()
	return nil
}
