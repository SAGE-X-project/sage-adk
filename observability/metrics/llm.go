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
	// LLM API metrics
	MetricLLMAPICalls      = "sage_llm_api_calls_total"
	MetricLLMAPIErrors     = "sage_llm_api_errors_total"
	MetricLLMAPILatency    = "sage_llm_api_latency_seconds"
	MetricLLMTokensTotal   = "sage_llm_tokens_total"
	MetricLLMTokensPrompt  = "sage_llm_tokens_prompt_total"
	MetricLLMTokensOutput  = "sage_llm_tokens_output_total"
	MetricLLMCostEstimated = "sage_llm_cost_estimated_usd"
)

// LLMMetrics provides LLM-specific metrics.
type LLMMetrics struct {
	collector Collector
}

// NewLLMMetrics creates a new LLM metrics collector.
func NewLLMMetrics(collector Collector) *LLMMetrics {
	return &LLMMetrics{
		collector: collector,
	}
}

// RecordCall records an LLM API call with latency.
func (m *LLMMetrics) RecordCall(provider, model string, latency float64) {
	labels := NewLabels("provider", provider, "model", model)
	m.collector.IncrementCounter(MetricLLMAPICalls, labels)
	m.collector.ObserveHistogram(MetricLLMAPILatency, latency, labels)
}

// RecordError records an LLM API error.
func (m *LLMMetrics) RecordError(provider, model, errorType string) {
	labels := NewLabels(
		"provider", provider,
		"model", model,
		"type", errorType,
	)
	m.collector.IncrementCounter(MetricLLMAPIErrors, labels)
}

// RecordTokens records token usage (prompt + completion).
func (m *LLMMetrics) RecordTokens(provider, model string, promptTokens, completionTokens int) {
	labels := NewLabels("provider", provider, "model", model)

	// Total tokens
	totalTokens := float64(promptTokens + completionTokens)
	m.collector.AddCounter(MetricLLMTokensTotal, totalTokens, labels)

	// Prompt tokens
	promptLabels := labels.With("type", "prompt")
	m.collector.AddCounter(MetricLLMTokensPrompt, float64(promptTokens), promptLabels)

	// Output tokens
	outputLabels := labels.With("type", "output")
	m.collector.AddCounter(MetricLLMTokensOutput, float64(completionTokens), outputLabels)
}

// RecordCost records estimated cost for an LLM call.
func (m *LLMMetrics) RecordCost(provider, model string, costUSD float64) {
	labels := NewLabels("provider", provider, "model", model)
	m.collector.AddCounter(MetricLLMCostEstimated, costUSD, labels)
}

// RecordCallWithTokens records a complete LLM call with tokens and latency.
func (m *LLMMetrics) RecordCallWithTokens(provider, model string, latency float64, promptTokens, completionTokens int) {
	m.RecordCall(provider, model, latency)
	m.RecordTokens(provider, model, promptTokens, completionTokens)
}

// RecordCallWithCost records a complete LLM call with cost estimation.
func (m *LLMMetrics) RecordCallWithCost(provider, model string, latency float64, promptTokens, completionTokens int, costUSD float64) {
	m.RecordCall(provider, model, latency)
	m.RecordTokens(provider, model, promptTokens, completionTokens)
	m.RecordCost(provider, model, costUSD)
}
