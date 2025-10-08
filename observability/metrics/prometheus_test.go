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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusCollector(t *testing.T) {
	collector := NewPrometheusCollector()

	if collector == nil {
		t.Fatal("NewPrometheusCollector() returned nil")
	}

	if collector.registry == nil {
		t.Error("registry should not be nil")
	}
}

func TestIncrementCounter(t *testing.T) {
	collector := NewPrometheusCollector()

	labels := map[string]string{"method": "GET", "status": "200"}
	collector.IncrementCounter("test_requests_total", labels)

	// Verify metric exists
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_requests_total") {
		t.Error("metric test_requests_total not found in output")
	}

	if !strings.Contains(body, `method="GET"`) {
		t.Error("label method=\"GET\" not found in output")
	}
}

func TestAddCounter(t *testing.T) {
	collector := NewPrometheusCollector()

	labels := map[string]string{"type": "test"}
	collector.AddCounter("test_counter", 5.5, labels)

	// Verify metric exists
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_counter") {
		t.Error("metric test_counter not found in output")
	}
}

func TestSetGauge(t *testing.T) {
	collector := NewPrometheusCollector()

	labels := map[string]string{"type": "test"}
	collector.SetGauge("test_gauge", 42.0, labels)

	// Verify metric exists
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_gauge") {
		t.Error("metric test_gauge not found in output")
	}

	if !strings.Contains(body, "42") {
		t.Error("gauge value 42 not found in output")
	}
}

func TestObserveHistogram(t *testing.T) {
	collector := NewPrometheusCollector()

	labels := map[string]string{"endpoint": "/api/test"}
	collector.ObserveHistogram("test_duration_seconds", 0.123, labels)

	// Verify metric exists
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_duration_seconds") {
		t.Error("metric test_duration_seconds not found in output")
	}

	// Histograms create _bucket, _sum, _count metrics
	if !strings.Contains(body, "test_duration_seconds_bucket") {
		t.Error("histogram bucket not found in output")
	}
}

func TestObserveSummary(t *testing.T) {
	collector := NewPrometheusCollector()

	labels := map[string]string{"operation": "test"}
	collector.ObserveSummary("test_summary", 0.456, labels)

	// Verify metric exists
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_summary") {
		t.Error("metric test_summary not found in output")
	}

	// Summaries create quantile metrics
	if !strings.Contains(body, "quantile") {
		t.Error("summary quantile not found in output")
	}
}

func TestHandler(t *testing.T) {
	collector := NewPrometheusCollector()
	handler := collector.Handler()

	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	// Add a metric first
	collector.IncrementCounter("test_metric", nil)

	// Test handler responds to requests
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Verify we got metrics output
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "test_metric") {
		t.Error("expected test_metric in response body")
	}
}

func TestConcurrentAccess(t *testing.T) {
	collector := NewPrometheusCollector()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			labels := map[string]string{"id": string(rune('0' + n))}
			for j := 0; j < 100; j++ {
				collector.IncrementCounter("concurrent_test", labels)
				collector.SetGauge("concurrent_gauge", float64(j), labels)
				collector.ObserveHistogram("concurrent_histogram", float64(j)/100.0, labels)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics exist
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "concurrent_test") {
		t.Error("concurrent_test metric not found")
	}
}

func TestPrometheusNoLabels(t *testing.T) {
	collector := NewPrometheusCollector()

	// Test with nil labels
	collector.IncrementCounter("no_labels_counter", nil)

	// Test with empty map
	collector.SetGauge("empty_labels_gauge", 123, map[string]string{})

	// Verify metrics exist
	handler := collector.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "no_labels_counter") {
		t.Error("no_labels_counter not found")
	}

	if !strings.Contains(body, "empty_labels_gauge") {
		t.Error("empty_labels_gauge not found")
	}
}
