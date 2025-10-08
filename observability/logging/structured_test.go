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
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewStructuredLogger(t *testing.T) {
	logger := NewStructuredLogger(LevelInfo)

	if logger == nil {
		t.Fatal("NewStructuredLogger returned nil")
	}

	if logger.level != LevelInfo {
		t.Errorf("expected level Info, got %s", logger.level)
	}

	if logger.samplingRate != 1.0 {
		t.Errorf("expected sampling rate 1.0, got %f", logger.samplingRate)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelDebug, &buf)

	ctx := context.Background()

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug",
			logFunc: func() {
				logger.Debug(ctx, "debug message")
			},
			expected: "debug",
		},
		{
			name: "info",
			logFunc: func() {
				logger.Info(ctx, "info message")
			},
			expected: "info",
		},
		{
			name: "warn",
			logFunc: func() {
				logger.Warn(ctx, "warn message")
			},
			expected: "warn",
		},
		{
			name: "error",
			logFunc: func() {
				logger.Error(ctx, "error message")
			},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain %s, got %s", tt.expected, output)
			}
		})
	}
}

func TestLogFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelWarn, &buf)

	ctx := context.Background()

	// Debug and Info should not be logged
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")

	if buf.Len() > 0 {
		t.Error("expected no output for debug/info when level is warn")
	}

	// Warn should be logged
	logger.Warn(ctx, "warn message")

	if buf.Len() == 0 {
		t.Error("expected output for warn message")
	}
}

func TestStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	ctx := context.Background()

	logger.Info(ctx, "test message",
		String("string_field", "value"),
		Int("int_field", 42),
		Float64("float_field", 3.14),
		Bool("bool_field", true),
	)

	// Parse JSON
	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["message"] != "test message" {
		t.Error("message field incorrect")
	}

	if entry["string_field"] != "value" {
		t.Error("string_field incorrect")
	}

	if entry["int_field"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("int_field incorrect: %v", entry["int_field"])
	}

	if entry["bool_field"] != true {
		t.Error("bool_field incorrect")
	}
}

func TestContextFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithAgentID(ctx, "agent-1")

	logger.Info(ctx, "test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["request_id"] != "req-123" {
		t.Error("request_id not found in log")
	}

	if entry["agent_id"] != "agent-1" {
		t.Error("agent_id not found in log")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	// Create child logger with persistent fields
	childLogger := logger.With(
		String("agent_id", "agent-1"),
		String("version", "1.0.0"),
	)

	ctx := context.Background()
	childLogger.Info(ctx, "test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["agent_id"] != "agent-1" {
		t.Error("agent_id not found in child logger output")
	}

	if entry["version"] != "1.0.0" {
		t.Error("version not found in child logger output")
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	ctx := context.Background()

	// Debug should not be logged at Info level
	logger.Debug(ctx, "debug message")
	if buf.Len() > 0 {
		t.Error("debug message logged at info level")
	}

	// Change level to Debug
	logger.SetLevel(LevelDebug)

	// Now debug should be logged
	logger.Debug(ctx, "debug message")
	if buf.Len() == 0 {
		t.Error("debug message not logged at debug level")
	}
}

func TestSampling(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelDebug, &buf)

	// Set low sampling rate
	logger.SetSamplingRate(0.0)

	ctx := context.Background()

	// Log many debug messages
	for i := 0; i < 100; i++ {
		logger.Debug(ctx, "debug message")
	}

	// With 0% sampling, no messages should be logged
	if buf.Len() > 0 {
		t.Error("expected no debug messages with 0% sampling")
	}

	// Set 100% sampling
	buf.Reset()
	logger.SetSamplingRate(1.0)

	logger.Debug(ctx, "debug message")

	if buf.Len() == 0 {
		t.Error("expected debug message with 100% sampling")
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	ctx := context.Background()
	logger.Info(ctx, "test message", String("key", "value"))

	// Verify valid JSON
	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"timestamp", "level", "message"}
	for _, field := range requiredFields {
		if _, ok := entry[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelInfo, &buf)

	ctx := context.Background()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				logger.Info(ctx, "concurrent message", Int("id", n))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify output contains messages (don't check exact count due to race)
	if buf.Len() == 0 {
		t.Error("expected log output from concurrent writes")
	}
}

func TestErrorField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLoggerWithOutput(LevelError, &buf)

	ctx := context.Background()
	testErr := &testError{msg: "test error"}

	logger.Error(ctx, "error occurred", Error(testErr))

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", entry["error"])
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
