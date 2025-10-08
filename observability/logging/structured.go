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
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"
)

// StructuredLogger is a JSON structured logger implementation.
type StructuredLogger struct {
	level        Level
	output       io.Writer
	fields       []Field
	samplingRate float64
	mu           sync.Mutex
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(level Level) *StructuredLogger {
	return &StructuredLogger{
		level:        level,
		output:       os.Stdout,
		fields:       []Field{},
		samplingRate: 1.0, // No sampling by default
	}
}

// NewStructuredLoggerWithOutput creates a logger with custom output.
func NewStructuredLoggerWithOutput(level Level, output io.Writer) *StructuredLogger {
	return &StructuredLogger{
		level:        level,
		output:       output,
		fields:       []Field{},
		samplingRate: 1.0,
	}
}

// Debug logs a debug message.
func (l *StructuredLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	if !l.shouldLog(LevelDebug) {
		return
	}

	// Apply sampling for debug logs
	if l.level == LevelDebug && l.samplingRate < 1.0 {
		if rand.Float64() > l.samplingRate {
			return
		}
	}

	l.log(ctx, LevelDebug, msg, fields...)
}

// Info logs an informational message.
func (l *StructuredLogger) Info(ctx context.Context, msg string, fields ...Field) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	l.log(ctx, LevelInfo, msg, fields...)
}

// Warn logs a warning message.
func (l *StructuredLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	if !l.shouldLog(LevelWarn) {
		return
	}
	l.log(ctx, LevelWarn, msg, fields...)
}

// Error logs an error message.
func (l *StructuredLogger) Error(ctx context.Context, msg string, fields ...Field) {
	if !l.shouldLog(LevelError) {
		return
	}
	l.log(ctx, LevelError, msg, fields...)
}

// Fatal logs a fatal message and exits.
func (l *StructuredLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelFatal, msg, fields...)
	os.Exit(1)
}

// With creates a child logger with persistent fields.
func (l *StructuredLogger) With(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &StructuredLogger{
		level:        l.level,
		output:       l.output,
		fields:       newFields,
		samplingRate: l.samplingRate,
	}
}

// SetLevel sets the minimum log level.
func (l *StructuredLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetSamplingRate sets the sampling rate for debug logs.
func (l *StructuredLogger) SetSamplingRate(rate float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}

	l.samplingRate = rate
}

// shouldLog checks if a message should be logged based on level.
func (l *StructuredLogger) shouldLog(level Level) bool {
	return levelPriority(level) >= levelPriority(l.level)
}

// log writes a log entry.
func (l *StructuredLogger) log(ctx context.Context, level Level, msg string, fields ...Field) {
	entry := l.buildEntry(ctx, level, msg, fields...)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.write(entry)
}

// buildEntry builds a log entry map.
func (l *StructuredLogger) buildEntry(ctx context.Context, level Level, msg string, fields ...Field) map[string]interface{} {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     string(level),
		"message":   msg,
	}

	// Add context fields
	contextFields := extractContextFields(ctx)
	for _, f := range contextFields {
		entry[f.Key] = f.Value
	}

	// Add logger persistent fields
	for _, f := range l.fields {
		entry[f.Key] = f.Value
	}

	// Add message fields
	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	return entry
}

// write writes the entry to output.
func (l *StructuredLogger) write(entry map[string]interface{}) {
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple error output
		data = []byte(`{"error":"failed to marshal log entry"}`)
	}

	l.output.Write(append(data, '\n'))
}
