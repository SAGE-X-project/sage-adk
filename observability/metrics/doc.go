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

// Package metrics provides metrics collection and export for SAGE ADK agents.
//
// # Overview
//
// This package provides a Prometheus-based metrics collector with support for:
//   - Counters (monotonic increasing values)
//   - Gauges (arbitrary values)
//   - Histograms (distribution of values)
//   - Summaries (quantiles)
//
// # Basic Usage
//
//	collector := metrics.NewPrometheusCollector()
//
//	// Increment counter
//	collector.IncrementCounter("requests_total", map[string]string{
//	    "method": "POST",
//	    "status": "200",
//	})
//
//	// Set gauge
//	collector.SetGauge("active_connections", 42, nil)
//
//	// Observe histogram
//	collector.ObserveHistogram("request_duration_seconds", 0.042, map[string]string{
//	    "endpoint": "/api/chat",
//	})
//
//	// Expose metrics
//	http.Handle("/metrics", collector.Handler())
//
// # Agent Metrics
//
// Pre-defined metrics for agent monitoring:
//
//	agentMetrics := metrics.NewAgentMetrics(collector)
//
//	// Record request
//	agentMetrics.RecordRequest("agent-1", "a2a", 0.042)
//
//	// Record error
//	agentMetrics.RecordError("agent-1", "llm_timeout")
//
//	// Update status
//	agentMetrics.SetStatus("agent-1", 1) // 1=healthy, 0=unhealthy
//
// # LLM Metrics
//
//	llmMetrics := metrics.NewLLMMetrics(collector)
//
//	// Record LLM call
//	llmMetrics.RecordCall("openai", "gpt-4", 0.523)
//
//	// Record token usage
//	llmMetrics.RecordTokens("openai", "gpt-4", 150, 450)
//
// # Custom Metrics
//
// Create custom metric collectors:
//
//	type CustomMetrics struct {
//	    collector metrics.Collector
//	}
//
//	func (m *CustomMetrics) RecordCustomEvent(name string) {
//	    m.collector.IncrementCounter("custom_events_total", map[string]string{
//	        "event": name,
//	    })
//	}
package metrics
