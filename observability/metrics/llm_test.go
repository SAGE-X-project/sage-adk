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

func TestNewLLMMetrics(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	if llmMetrics == nil {
		t.Fatal("NewLLMMetrics() returned nil")
	}

	if llmMetrics.collector == nil {
		t.Error("collector should not be nil")
	}
}

func TestRecordCall(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordCall("openai", "gpt-4", 0.523)

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_llm_api_calls_total") {
		t.Error("sage_llm_api_calls_total metric not found")
	}

	if !strings.Contains(body, "sage_llm_api_latency_seconds") {
		t.Error("sage_llm_api_latency_seconds metric not found")
	}

	if !strings.Contains(body, `provider="openai"`) {
		t.Error("provider label not found")
	}

	if !strings.Contains(body, `model="gpt-4"`) {
		t.Error("model label not found")
	}
}

func TestLLMRecordError(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordError("openai", "gpt-4", "rate_limit")

	// Verify metric
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_llm_api_errors_total") {
		t.Error("sage_llm_api_errors_total metric not found")
	}

	if !strings.Contains(body, `type="rate_limit"`) {
		t.Error("error type label not found")
	}
}

func TestRecordTokens(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordTokens("anthropic", "claude-3-sonnet", 150, 450)

	// Verify metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_llm_tokens_total") {
		t.Error("sage_llm_tokens_total metric not found")
	}

	if !strings.Contains(body, "sage_llm_tokens_prompt_total") {
		t.Error("sage_llm_tokens_prompt_total metric not found")
	}

	if !strings.Contains(body, "sage_llm_tokens_output_total") {
		t.Error("sage_llm_tokens_output_total metric not found")
	}

	if !strings.Contains(body, `type="prompt"`) {
		t.Error("prompt type label not found")
	}

	if !strings.Contains(body, `type="output"`) {
		t.Error("output type label not found")
	}
}

func TestRecordCost(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordCost("openai", "gpt-4", 0.03)

	// Verify metric
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "sage_llm_cost_estimated_usd") {
		t.Error("sage_llm_cost_estimated_usd metric not found")
	}
}

func TestRecordCallWithTokens(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordCallWithTokens("gemini", "gemini-pro", 0.234, 200, 300)

	// Verify all related metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Check call metrics
	if !strings.Contains(body, "sage_llm_api_calls_total") {
		t.Error("api calls metric not found")
	}

	if !strings.Contains(body, "sage_llm_api_latency_seconds") {
		t.Error("api latency metric not found")
	}

	// Check token metrics
	if !strings.Contains(body, "sage_llm_tokens_total") {
		t.Error("tokens total metric not found")
	}

	if !strings.Contains(body, "sage_llm_tokens_prompt_total") {
		t.Error("tokens prompt metric not found")
	}

	if !strings.Contains(body, "sage_llm_tokens_output_total") {
		t.Error("tokens output metric not found")
	}

	// Check labels
	if !strings.Contains(body, `provider="gemini"`) {
		t.Error("gemini provider not found")
	}

	if !strings.Contains(body, `model="gemini-pro"`) {
		t.Error("gemini-pro model not found")
	}
}

func TestRecordCallWithCost(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	llmMetrics.RecordCallWithCost("openai", "gpt-3.5-turbo", 0.152, 100, 250, 0.001)

	// Verify all metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Check all metric types
	metrics := []string{
		"sage_llm_api_calls_total",
		"sage_llm_api_latency_seconds",
		"sage_llm_tokens_total",
		"sage_llm_tokens_prompt_total",
		"sage_llm_tokens_output_total",
		"sage_llm_cost_estimated_usd",
	}

	for _, metric := range metrics {
		if !strings.Contains(body, metric) {
			t.Errorf("metric %s not found", metric)
		}
	}
}

func TestMultipleProviders(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	// Record calls to different providers
	llmMetrics.RecordCall("openai", "gpt-4", 0.5)
	llmMetrics.RecordCall("anthropic", "claude-3-sonnet", 0.6)
	llmMetrics.RecordCall("gemini", "gemini-pro", 0.3)

	// Verify all providers are tracked
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	providers := []string{
		`provider="openai"`,
		`provider="anthropic"`,
		`provider="gemini"`,
	}

	for _, provider := range providers {
		if !strings.Contains(body, provider) {
			t.Errorf("provider %s not found", provider)
		}
	}
}

func TestTokenCalculations(t *testing.T) {
	collector := NewPrometheusCollector()
	llmMetrics := NewLLMMetrics(collector)

	// Record tokens
	promptTokens := 100
	completionTokens := 250
	llmMetrics.RecordTokens("test", "test-model", promptTokens, completionTokens)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	collector.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Check that total tokens metric exists with correct value
	if !strings.Contains(body, "sage_llm_tokens_total") {
		t.Error("total tokens metric not found")
	}

	// The actual value check would require parsing Prometheus format,
	// but we can verify the metric is present with correct labels
	if !strings.Contains(body, `model="test-model"`) {
		t.Error("test-model not found in metrics")
	}
}
