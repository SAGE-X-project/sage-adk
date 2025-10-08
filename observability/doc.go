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

// Package observability provides monitoring, logging, and tracing capabilities
// for SAGE ADK agents.
//
// # Overview
//
// This package enables comprehensive observability for AI agents through:
//   - Metrics collection (Prometheus)
//   - Structured logging
//   - Distributed tracing (OpenTelemetry)
//   - Health checks
//
// # Metrics
//
// Collect and expose metrics for monitoring:
//
//	collector := metrics.NewPrometheusCollector()
//	agentMetrics := metrics.NewAgentMetrics(collector)
//
//	// Record request
//	agentMetrics.RecordRequest("agent-1", "a2a", 0.042)
//
//	// Expose metrics
//	http.Handle("/metrics", collector.Handler())
//
// # Logging
//
// Structured logging with context propagation:
//
//	logger := logging.NewStructuredLogger(logging.LevelInfo)
//
//	ctx := logging.WithRequestID(ctx, "req-123")
//	logger.Info(ctx, "message handled",
//	    logging.String("agent_id", "agent-1"),
//	    logging.Int("duration_ms", 42),
//	)
//
// # Tracing
//
// Distributed tracing with OpenTelemetry:
//
//	tracer := tracing.NewOTelTracer(config)
//	defer tracer.Shutdown(ctx)
//
//	ctx, span := tracer.Start(ctx, "handle_message")
//	defer span.End()
//
// # Health Checks
//
// Liveness, readiness, and startup probes:
//
//	liveness := health.NewLivenessChecker()
//	readiness := health.NewReadinessChecker(
//	    health.NewLLMHealthCheck(provider),
//	    health.NewStorageHealthCheck(storage),
//	)
//
//	http.Handle("/health/live", health.Handler(liveness))
//	http.Handle("/health/ready", health.Handler(readiness))
//
// # Integration with Agent
//
//	agent := builder.NewAgent("monitored-agent").
//	    WithObservability(&observability.Config{
//	        Metrics: observability.MetricsConfig{Enabled: true},
//	        Logging: observability.LoggingConfig{Level: "info"},
//	        Tracing: observability.TracingConfig{Enabled: true},
//	    }).
//	    Build()
package observability
