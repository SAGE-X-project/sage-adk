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

import (
	"context"
	"testing"
)

func TestRequestID(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := GetRequestID(ctx); id != "" {
		t.Errorf("expected empty request ID, got %s", id)
	}

	// Test with request ID
	ctx = WithRequestID(ctx, "req-123")
	if id := GetRequestID(ctx); id != "req-123" {
		t.Errorf("expected request ID 'req-123', got %s", id)
	}
}

func TestTraceID(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := GetTraceID(ctx); id != "" {
		t.Errorf("expected empty trace ID, got %s", id)
	}

	// Test with trace ID
	ctx = WithTraceID(ctx, "trace-456")
	if id := GetTraceID(ctx); id != "trace-456" {
		t.Errorf("expected trace ID 'trace-456', got %s", id)
	}
}

func TestSpanID(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := GetSpanID(ctx); id != "" {
		t.Errorf("expected empty span ID, got %s", id)
	}

	// Test with span ID
	ctx = WithSpanID(ctx, "span-789")
	if id := GetSpanID(ctx); id != "span-789" {
		t.Errorf("expected span ID 'span-789', got %s", id)
	}
}

func TestAgentID(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := GetAgentID(ctx); id != "" {
		t.Errorf("expected empty agent ID, got %s", id)
	}

	// Test with agent ID
	ctx = WithAgentID(ctx, "agent-1")
	if id := GetAgentID(ctx); id != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %s", id)
	}
}

func TestUserID(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := GetUserID(ctx); id != "" {
		t.Errorf("expected empty user ID, got %s", id)
	}

	// Test with user ID
	ctx = WithUserID(ctx, "user-42")
	if id := GetUserID(ctx); id != "user-42" {
		t.Errorf("expected user ID 'user-42', got %s", id)
	}
}

func TestExtractContextFields(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	fields := extractContextFields(ctx)
	if len(fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(fields))
	}

	// Test with all IDs
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")
	ctx = WithSpanID(ctx, "span-789")
	ctx = WithAgentID(ctx, "agent-1")
	ctx = WithUserID(ctx, "user-42")

	fields = extractContextFields(ctx)

	if len(fields) != 5 {
		t.Errorf("expected 5 fields, got %d", len(fields))
	}

	// Verify field values
	fieldMap := make(map[string]interface{})
	for _, f := range fields {
		fieldMap[f.Key] = f.Value
	}

	if fieldMap["request_id"] != "req-123" {
		t.Error("request_id field incorrect")
	}

	if fieldMap["trace_id"] != "trace-456" {
		t.Error("trace_id field incorrect")
	}

	if fieldMap["span_id"] != "span-789" {
		t.Error("span_id field incorrect")
	}

	if fieldMap["agent_id"] != "agent-1" {
		t.Error("agent_id field incorrect")
	}

	if fieldMap["user_id"] != "user-42" {
		t.Error("user_id field incorrect")
	}
}

func TestPartialContextFields(t *testing.T) {
	ctx := context.Background()

	// Test with only some IDs
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithAgentID(ctx, "agent-1")

	fields := extractContextFields(ctx)

	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}

	fieldMap := make(map[string]interface{})
	for _, f := range fields {
		fieldMap[f.Key] = f.Value
	}

	if fieldMap["request_id"] != "req-123" {
		t.Error("request_id field incorrect")
	}

	if fieldMap["agent_id"] != "agent-1" {
		t.Error("agent_id field incorrect")
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	// Chain context additions
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTraceID(ctx, "trace-1")
	ctx = WithAgentID(ctx, "agent-1")

	// Verify all values are preserved
	if GetRequestID(ctx) != "req-1" {
		t.Error("request ID not preserved in chaining")
	}

	if GetTraceID(ctx) != "trace-1" {
		t.Error("trace ID not preserved in chaining")
	}

	if GetAgentID(ctx) != "agent-1" {
		t.Error("agent ID not preserved in chaining")
	}
}
