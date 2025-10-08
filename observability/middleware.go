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
	"net/http"
	"time"

	"github.com/sage-x-project/sage-adk/observability/logging"
	"github.com/sage-x-project/sage-adk/observability/metrics"
)

// Middleware provides HTTP middleware for observability.
type Middleware struct {
	logger  logging.Logger
	metrics *metrics.AgentMetrics
	agentID string
}

// NewMiddleware creates a new observability middleware.
func NewMiddleware(logger logging.Logger, m *metrics.AgentMetrics, agentID string) *Middleware {
	return &Middleware{
		logger:  logger,
		metrics: m,
		agentID: agentID,
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Handler returns an HTTP middleware that logs requests and records metrics.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create context with request ID
		ctx := r.Context()
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" {
			ctx = logging.WithRequestID(ctx, requestID)
		}

		// Add agent ID to context
		ctx = logging.WithAgentID(ctx, m.agentID)
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log request
		m.logger.Info(ctx, "incoming request",
			logging.String("method", r.Method),
			logging.String("path", r.URL.Path),
			logging.String("remote_addr", r.RemoteAddr),
		)

		// Serve request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record metrics
		protocol := "http"
		if r.TLS != nil {
			protocol = "https"
		}
		m.metrics.RecordRequest(m.agentID, protocol, duration)

		// Record error if non-2xx status
		if rw.statusCode >= 400 {
			errorType := "client_error"
			if rw.statusCode >= 500 {
				errorType = "server_error"
			}
			m.metrics.RecordError(m.agentID, errorType)

			m.logger.Error(ctx, "request error",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.Int("status", rw.statusCode),
				logging.Float64("duration_sec", duration),
			)
		} else {
			m.logger.Info(ctx, "request completed",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.Int("status", rw.statusCode),
				logging.Float64("duration_sec", duration),
				logging.Int("bytes_written", int(rw.written)),
			)
		}
	})
}

// HandlerFunc returns an HTTP middleware that can wrap http.HandlerFunc.
func (m *Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler(next).ServeHTTP(w, r)
	}
}
