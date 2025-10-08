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

package logging

import "context"

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
	spanIDKey    contextKey = "span_id"
	agentIDKey   contextKey = "agent_id"
	userIDKey    contextKey = "user_id"
)

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithSpanID adds a span ID to the context.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// GetSpanID retrieves the span ID from the context.
func GetSpanID(ctx context.Context) string {
	if v := ctx.Value(spanIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithAgentID adds an agent ID to the context.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

// GetAgentID retrieves the agent ID from the context.
func GetAgentID(ctx context.Context) string {
	if v := ctx.Value(agentIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractContextFields extracts all known context fields.
func extractContextFields(ctx context.Context) []Field {
	fields := make([]Field, 0, 5)

	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, String("request_id", requestID))
	}

	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}

	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, String("span_id", spanID))
	}

	if agentID := GetAgentID(ctx); agentID != "" {
		fields = append(fields, String("agent_id", agentID))
	}

	if userID := GetUserID(ctx); userID != "" {
		fields = append(fields, String("user_id", userID))
	}

	return fields
}
