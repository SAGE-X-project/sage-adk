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

package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Handler creates an HTTP handler for a health check.
func Handler(checker Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set timeout for health check
		timeout := 5 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
		}

		// Create context with timeout
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Perform health check
		result := checker.Check(checkCtx)

		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Set status code based on health
		statusCode := http.StatusOK
		if result.IsUnhealthy() {
			statusCode = http.StatusServiceUnavailable
		} else if result.IsDegraded() {
			statusCode = http.StatusOK // Still return 200 for degraded
		}

		w.WriteHeader(statusCode)

		// Write JSON response
		if err := json.NewEncoder(w).Encode(result); err != nil {
			// Fallback error response
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "failed to encode health check result",
			})
		}
	}
}

// MultiHandler creates an HTTP handler that runs multiple health checks.
func MultiHandler(checkers ...Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set timeout for health checks
		timeout := 10 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
		}

		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		results := make([]CheckResult, 0, len(checkers))
		overallStatus := StatusHealthy

		for _, checker := range checkers {
			result := checker.Check(checkCtx)
			results = append(results, result)

			// Determine overall status
			if result.IsUnhealthy() {
				overallStatus = StatusUnhealthy
			} else if result.IsDegraded() && overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}

		response := map[string]interface{}{
			"status": overallStatus,
			"checks": results,
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Set status code
		statusCode := http.StatusOK
		if overallStatus == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)

		// Write JSON response
		json.NewEncoder(w).Encode(response)
	}
}
