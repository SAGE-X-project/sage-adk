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

package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAgentMetrics(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	if agentMetrics == nil {
		t.Fatal("NewAgentMetrics() returned nil")
	}

	if agentMetrics.collector == nil {
		t.Error("collector should not be nil")
	}
}

func TestSetStatus(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.SetStatus("agent-1", 1)

	// Verify metric
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "sage_agent_status") {
		t.Error("sage_agent_status metric not found")
	}

	if !strings.Contains(body, `agent_id="agent-1"`) {
		t.Error("agent_id label not found")
	}
}

func TestRecordRequest(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.RecordRequest("agent-1", "a2a", 0.042)

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_agent_requests_total") {
		t.Error("sage_agent_requests_total metric not found")
	}

	if !strings.Contains(body, "sage_agent_request_duration_seconds") {
		t.Error("sage_agent_request_duration_seconds metric not found")
	}

	if !strings.Contains(body, `protocol="a2a"`) {
		t.Error("protocol label not found")
	}
}

func TestRecordError(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.RecordError("agent-1", "timeout")

	// Verify metric
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_agent_errors_total") {
		t.Error("sage_agent_errors_total metric not found")
	}

	if !strings.Contains(body, `type="timeout"`) {
		t.Error("error type label not found")
	}
}

func TestRecordMessages(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.RecordMessageReceived("agent-1", "a2a", "request")
	agentMetrics.RecordMessageSent("agent-1", "a2a", "response")

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_agent_messages_received_total") {
		t.Error("sage_agent_messages_received_total metric not found")
	}

	if !strings.Contains(body, "sage_agent_messages_sent_total") {
		t.Error("sage_agent_messages_sent_total metric not found")
	}

	if !strings.Contains(body, `type="request"`) {
		t.Error("message type label not found")
	}
}

func TestSetSystemMetrics(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.SetActiveGoroutines("agent-1", 100)
	agentMetrics.SetMemoryUsage("agent-1", 1024*1024*512) // 512 MB
	agentMetrics.SetCPUUsage("agent-1", 45.5)

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_agent_active_goroutines") {
		t.Error("sage_agent_active_goroutines metric not found")
	}

	if !strings.Contains(body, "sage_agent_memory_bytes") {
		t.Error("sage_agent_memory_bytes metric not found")
	}

	if !strings.Contains(body, "sage_agent_cpu_usage_percent") {
		t.Error("sage_agent_cpu_usage_percent metric not found")
	}

	if !strings.Contains(body, "100") {
		t.Error("goroutines value not found")
	}
}

func TestProtocolMetrics(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	agentMetrics.RecordProtocolRequest("agent-1", "sage", "handshake")
	agentMetrics.RecordProtocolError("agent-1", "sage", "signature_failed")
	agentMetrics.RecordHandshakeTime("agent-1", 0.123)
	agentMetrics.RecordSigningTime("agent-1", 0.005)
	agentMetrics.RecordVerificationTime("agent-1", 0.003)

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_agent_protocol_requests_total") {
		t.Error("sage_agent_protocol_requests_total metric not found")
	}

	if !strings.Contains(body, "sage_agent_protocol_errors_total") {
		t.Error("sage_agent_protocol_errors_total metric not found")
	}

	if !strings.Contains(body, "sage_agent_handshake_duration_seconds") {
		t.Error("sage_agent_handshake_duration_seconds metric not found")
	}

	if !strings.Contains(body, "sage_agent_signing_duration_seconds") {
		t.Error("sage_agent_signing_duration_seconds metric not found")
	}

	if !strings.Contains(body, "sage_agent_verification_duration_seconds") {
		t.Error("sage_agent_verification_duration_seconds metric not found")
	}

	if !strings.Contains(body, `operation="handshake"`) {
		t.Error("operation label not found")
	}

	if !strings.Contains(body, `type="signature_failed"`) {
		t.Error("error type label not found")
	}
}

func TestMultipleAgents(t *testing.T) {
	collector := NewPrometheusCollector()
	agentMetrics := NewAgentMetrics(collector)

	// Record metrics for multiple agents
	agentMetrics.RecordRequest("agent-1", "a2a", 0.01)
	agentMetrics.RecordRequest("agent-2", "sage", 0.02)
	agentMetrics.RecordRequest("agent-3", "a2a", 0.03)

	// Verify all agents are tracked
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, `agent_id="agent-1"`) {
		t.Error("agent-1 not found")
	}

	if !strings.Contains(body, `agent_id="agent-2"`) {
		t.Error("agent-2 not found")
	}

	if !strings.Contains(body, `agent_id="agent-3"`) {
		t.Error("agent-3 not found")
	}
}
