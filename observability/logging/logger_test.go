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
	"errors"
	"testing"
)

func TestString(t *testing.T) {
	field := String("key", "value")

	if field.Key != "key" {
		t.Errorf("expected key 'key', got %s", field.Key)
	}

	if field.Value != "value" {
		t.Errorf("expected value 'value', got %v", field.Value)
	}
}

func TestInt(t *testing.T) {
	field := Int("count", 42)

	if field.Key != "count" {
		t.Errorf("expected key 'count', got %s", field.Key)
	}

	if field.Value != 42 {
		t.Errorf("expected value 42, got %v", field.Value)
	}
}

func TestFloat64(t *testing.T) {
	field := Float64("duration", 0.523)

	if field.Key != "duration" {
		t.Errorf("expected key 'duration', got %s", field.Key)
	}

	if field.Value != 0.523 {
		t.Errorf("expected value 0.523, got %v", field.Value)
	}
}

func TestBool(t *testing.T) {
	field := Bool("success", true)

	if field.Key != "success" {
		t.Errorf("expected key 'success', got %s", field.Key)
	}

	if field.Value != true {
		t.Errorf("expected value true, got %v", field.Value)
	}
}

func TestError(t *testing.T) {
	err := errors.New("test error")
	field := Error(err)

	if field.Key != "error" {
		t.Errorf("expected key 'error', got %s", field.Key)
	}

	if field.Value != "test error" {
		t.Errorf("expected value 'test error', got %v", field.Value)
	}

	// Test nil error
	nilField := Error(nil)
	if nilField.Value != nil {
		t.Errorf("expected nil value for nil error, got %v", nilField.Value)
	}
}

func TestAny(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	field := Any("data", data)

	if field.Key != "data" {
		t.Errorf("expected key 'data', got %s", field.Key)
	}

	if field.Value == nil {
		t.Error("expected non-nil value")
	}
}

func TestFields(t *testing.T) {
	fields := Fields("key1", "value1", "key2", 42)

	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}

	if fields[0].Key != "key1" || fields[0].Value != "value1" {
		t.Error("first field incorrect")
	}

	if fields[1].Key != "key2" || fields[1].Value != 42 {
		t.Error("second field incorrect")
	}
}

func TestFieldsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for odd number of arguments")
		}
	}()

	Fields("key")
}

func TestFieldsNonStringKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-string key")
		}
	}()

	Fields(123, "value")
}

func TestLevelPriority(t *testing.T) {
	tests := []struct {
		level    Level
		priority int
	}{
		{LevelDebug, 0},
		{LevelInfo, 1},
		{LevelWarn, 2},
		{LevelError, 3},
		{LevelFatal, 4},
		{Level("unknown"), 1}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			priority := levelPriority(tt.level)
			if priority != tt.priority {
				t.Errorf("expected priority %d for level %s, got %d", tt.priority, tt.level, priority)
			}
		})
	}
}

func TestDuration(t *testing.T) {
	field := Duration("elapsed", 42)

	if field.Key != "elapsed" {
		t.Errorf("expected key 'elapsed', got %s", field.Key)
	}

	if field.Value != int64(42) {
		t.Errorf("expected value 42, got %v", field.Value)
	}
}

func TestInt64(t *testing.T) {
	field := Int64("timestamp", 1234567890)

	if field.Key != "timestamp" {
		t.Errorf("expected key 'timestamp', got %s", field.Key)
	}

	if field.Value != int64(1234567890) {
		t.Errorf("expected value 1234567890, got %v", field.Value)
	}
}
