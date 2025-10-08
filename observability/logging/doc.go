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

// Package logging provides structured logging with context propagation for SAGE ADK agents.
//
// # Overview
//
// This package provides structured logging with:
//   - Multiple log levels (DEBUG, INFO, WARN, ERROR, FATAL)
//   - JSON and text output formats
//   - Context-aware logging (request ID, trace ID, agent ID)
//   - Log sampling for high-volume scenarios
//   - Field-based structured data
//
// # Basic Usage
//
//	logger := logging.NewStructuredLogger(logging.LevelInfo)
//
//	logger.Info(ctx, "message handled",
//	    logging.String("agent_id", "agent-1"),
//	    logging.Int("duration_ms", 42),
//	)
//
// # Context Propagation
//
// Automatically extract context values:
//
//	ctx = logging.WithRequestID(ctx, "req-123")
//	ctx = logging.WithTraceID(ctx, "trace-456")
//	ctx = logging.WithAgentID(ctx, "agent-1")
//
//	logger.Info(ctx, "processing request")
//	// Output: {"timestamp":"...","level":"info","message":"processing request","request_id":"req-123","trace_id":"trace-456","agent_id":"agent-1"}
//
// # Log Levels
//
//	logger.Debug(ctx, "detailed debug info")
//	logger.Info(ctx, "informational message")
//	logger.Warn(ctx, "warning message")
//	logger.Error(ctx, "error occurred", logging.Error(err))
//	logger.Fatal(ctx, "fatal error")  // Calls os.Exit(1)
//
// # Structured Fields
//
//	logger.Info(ctx, "user action",
//	    logging.String("user_id", "user-123"),
//	    logging.Int("count", 42),
//	    logging.Float64("duration", 0.523),
//	    logging.Bool("success", true),
//	    logging.Error(err),
//	    logging.Any("data", complexObject),
//	)
//
// # Log Sampling
//
// Sample debug logs for performance:
//
//	logger := logging.NewStructuredLogger(logging.LevelDebug)
//	logger.SetSamplingRate(0.1)  // Sample 10% of debug logs
//
//	for i := 0; i < 1000; i++ {
//	    logger.Debug(ctx, "debug message")  // Only ~100 will be logged
//	}
//
// # With Fields
//
// Add persistent fields to all logs:
//
//	agentLogger := logger.With(
//	    logging.String("agent_id", "agent-1"),
//	    logging.String("version", "1.0.0"),
//	)
//
//	agentLogger.Info(ctx, "started")   // Includes agent_id and version
//	agentLogger.Info(ctx, "stopped")   // Includes agent_id and version
//
// # Output Formats
//
// JSON (default):
//
//	{"timestamp":"2025-10-08T10:30:00Z","level":"info","message":"hello","agent_id":"agent-1"}
//
// Text:
//
//	2025-10-08T10:30:00Z INFO hello agent_id=agent-1
package logging
