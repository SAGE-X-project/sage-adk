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

// Level represents the log level.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelFatal Level = "fatal"
)

// Logger is the interface for structured logging.
type Logger interface {
	// Debug logs a debug message
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs an informational message
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs a warning message
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs an error message
	Error(ctx context.Context, msg string, fields ...Field)

	// Fatal logs a fatal message and exits
	Fatal(ctx context.Context, msg string, fields ...Field)

	// With creates a child logger with persistent fields
	With(fields ...Field) Logger

	// SetLevel sets the minimum log level
	SetLevel(level Level)

	// SetSamplingRate sets the sampling rate for debug logs (0.0-1.0)
	SetSamplingRate(rate float64)
}

// Field represents a structured log field.
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field.
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field (in milliseconds).
func Duration(key string, ms int64) Field {
	return Field{Key: key, Value: ms}
}

// Fields creates multiple fields at once.
func Fields(keyvals ...interface{}) []Field {
	if len(keyvals)%2 != 0 {
		panic("Fields requires an even number of arguments")
	}

	fields := make([]Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			panic("Field key must be a string")
		}
		fields = append(fields, Field{Key: key, Value: keyvals[i+1]})
	}

	return fields
}

// levelPriority returns the numeric priority of a log level.
func levelPriority(level Level) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	case LevelFatal:
		return 4
	default:
		return 1 // default to info
	}
}
