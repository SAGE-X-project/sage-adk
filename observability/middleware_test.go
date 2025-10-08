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
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sage-x-project/sage-adk/observability/logging"
	"github.com/sage-x-project/sage-adk/observability/metrics"
)

func TestNewMiddleware(t *testing.T) {
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	collector := metrics.NewPrometheusCollector()
	agentMetrics := metrics.NewAgentMetrics(collector)

	middleware := NewMiddleware(logger, agentMetrics, "test-agent")

	if middleware == nil {
		t.Fatal("expected non-nil middleware")
	}
	if middleware.agentID != "test-agent" {
		t.Errorf("expected agentID %s, got %s", "test-agent", middleware.agentID)
	}
}

func TestMiddleware_Handler_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewStructuredLoggerWithOutput(logging.LevelInfo, &buf)

	collector := metrics.NewPrometheusCollector()
	agentMetrics := metrics.NewAgentMetrics(collector)
	middleware := NewMiddleware(logger, agentMetrics, "test-agent")

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrapped := middleware.Handler(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-123")
	rec := httptest.NewRecorder()

	// Serve request
	wrapped.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify logs were written
	if buf.Len() == 0 {
		t.Error("expected logs to be written")
	}
}

func TestMiddleware_Handler_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewStructuredLoggerWithOutput(logging.LevelInfo, &buf)

	collector := metrics.NewPrometheusCollector()
	agentMetrics := metrics.NewAgentMetrics(collector)
	middleware := NewMiddleware(logger, agentMetrics, "test-agent")

	// Create test handler that returns error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	// Wrap with middleware
	wrapped := middleware.Handler(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Serve request
	wrapped.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Verify error logs were written
	logs := buf.String()
	if logs == "" {
		t.Error("expected error logs to be written")
	}
}

func TestMiddleware_Handler_ClientError(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewStructuredLoggerWithOutput(logging.LevelInfo, &buf)

	collector := metrics.NewPrometheusCollector()
	agentMetrics := metrics.NewAgentMetrics(collector)
	middleware := NewMiddleware(logger, agentMetrics, "test-agent")

	// Create test handler that returns client error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	})

	// Wrap with middleware
	wrapped := middleware.Handler(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	// Serve request
	wrapped.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestMiddleware_HandlerFunc(t *testing.T) {
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	collector := metrics.NewPrometheusCollector()
	agentMetrics := metrics.NewAgentMetrics(collector)
	middleware := NewMiddleware(logger, agentMetrics, "test-agent")

	// Create test handler func
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}

	// Wrap with middleware
	wrapped := middleware.HandlerFunc(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Serve request
	wrapped(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", rec.Body.String())
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rw.statusCode)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected recorder status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}
	if rw.written != int64(len(data)) {
		t.Errorf("expected %d bytes tracked, got %d", len(data), rw.written)
	}
	if rec.Body.String() != string(data) {
		t.Errorf("expected body '%s', got '%s'", string(data), rec.Body.String())
	}
}
