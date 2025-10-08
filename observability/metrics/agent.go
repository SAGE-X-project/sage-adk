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

const (
	// Agent status metrics
	MetricAgentStatus = "sage_agent_status"

	// Request metrics
	MetricRequestsTotal    = "sage_agent_requests_total"
	MetricRequestDuration  = "sage_agent_request_duration_seconds"
	MetricErrorsTotal      = "sage_agent_errors_total"
	MetricMessagesReceived = "sage_agent_messages_received_total"
	MetricMessagesSent     = "sage_agent_messages_sent_total"

	// System metrics
	MetricActiveGoroutines = "sage_agent_active_goroutines"
	MetricMemoryUsage      = "sage_agent_memory_bytes"
	MetricCPUUsage         = "sage_agent_cpu_usage_percent"

	// Protocol metrics
	MetricProtocolRequests  = "sage_agent_protocol_requests_total"
	MetricProtocolErrors    = "sage_agent_protocol_errors_total"
	MetricHandshakeTime     = "sage_agent_handshake_duration_seconds"
	MetricSigningTime       = "sage_agent_signing_duration_seconds"
	MetricVerificationTime  = "sage_agent_verification_duration_seconds"
)

// AgentMetrics provides agent-specific metrics.
type AgentMetrics struct {
	collector Collector
}

// NewAgentMetrics creates a new agent metrics collector.
func NewAgentMetrics(collector Collector) *AgentMetrics {
	return &AgentMetrics{
		collector: collector,
	}
}

// SetStatus sets the agent status (1=healthy, 0=unhealthy).
func (m *AgentMetrics) SetStatus(agentID string, status float64) {
	m.collector.SetGauge(MetricAgentStatus, status, NewLabels("agent_id", agentID))
}

// RecordRequest records a request with duration.
func (m *AgentMetrics) RecordRequest(agentID, protocol string, duration float64) {
	labels := NewLabels("agent_id", agentID, "protocol", protocol)
	m.collector.IncrementCounter(MetricRequestsTotal, labels)
	m.collector.ObserveHistogram(MetricRequestDuration, duration, labels)
}

// RecordError records an error.
func (m *AgentMetrics) RecordError(agentID, errorType string) {
	labels := NewLabels("agent_id", agentID, "type", errorType)
	m.collector.IncrementCounter(MetricErrorsTotal, labels)
}

// RecordMessageReceived records a received message.
func (m *AgentMetrics) RecordMessageReceived(agentID, protocol, messageType string) {
	labels := NewLabels(
		"agent_id", agentID,
		"protocol", protocol,
		"type", messageType,
	)
	m.collector.IncrementCounter(MetricMessagesReceived, labels)
}

// RecordMessageSent records a sent message.
func (m *AgentMetrics) RecordMessageSent(agentID, protocol, messageType string) {
	labels := NewLabels(
		"agent_id", agentID,
		"protocol", protocol,
		"type", messageType,
	)
	m.collector.IncrementCounter(MetricMessagesSent, labels)
}

// SetActiveGoroutines sets the number of active goroutines.
func (m *AgentMetrics) SetActiveGoroutines(agentID string, count float64) {
	m.collector.SetGauge(MetricActiveGoroutines, count, NewLabels("agent_id", agentID))
}

// SetMemoryUsage sets the memory usage in bytes.
func (m *AgentMetrics) SetMemoryUsage(agentID string, bytes float64) {
	m.collector.SetGauge(MetricMemoryUsage, bytes, NewLabels("agent_id", agentID))
}

// SetCPUUsage sets the CPU usage percentage.
func (m *AgentMetrics) SetCPUUsage(agentID string, percent float64) {
	m.collector.SetGauge(MetricCPUUsage, percent, NewLabels("agent_id", agentID))
}

// RecordProtocolRequest records a protocol-specific request.
func (m *AgentMetrics) RecordProtocolRequest(agentID, protocol, operation string) {
	labels := NewLabels(
		"agent_id", agentID,
		"protocol", protocol,
		"operation", operation,
	)
	m.collector.IncrementCounter(MetricProtocolRequests, labels)
}

// RecordProtocolError records a protocol-specific error.
func (m *AgentMetrics) RecordProtocolError(agentID, protocol, errorType string) {
	labels := NewLabels(
		"agent_id", agentID,
		"protocol", protocol,
		"type", errorType,
	)
	m.collector.IncrementCounter(MetricProtocolErrors, labels)
}

// RecordHandshakeTime records SAGE handshake duration.
func (m *AgentMetrics) RecordHandshakeTime(agentID string, duration float64) {
	labels := NewLabels("agent_id", agentID)
	m.collector.ObserveHistogram(MetricHandshakeTime, duration, labels)
}

// RecordSigningTime records message signing duration.
func (m *AgentMetrics) RecordSigningTime(agentID string, duration float64) {
	labels := NewLabels("agent_id", agentID)
	m.collector.ObserveHistogram(MetricSigningTime, duration, labels)
}

// RecordVerificationTime records signature verification duration.
func (m *AgentMetrics) RecordVerificationTime(agentID string, duration float64) {
	labels := NewLabels("agent_id", agentID)
	m.collector.ObserveHistogram(MetricVerificationTime, duration, labels)
}
