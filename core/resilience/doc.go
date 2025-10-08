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

// Package resilience provides resilience patterns for building fault-tolerant systems.
//
// Resilience patterns help systems handle failures gracefully and recover automatically.
// This package includes:
//   - Retry: Automatically retry failed operations
//   - Circuit Breaker: Prevent cascading failures
//   - Bulkhead: Isolate resources to prevent resource exhaustion
//   - Timeout: Set time limits on operations
//
// Retry Pattern:
//
//	config := &resilience.RetryConfig{
//	    MaxAttempts: 3,
//	    Backoff:     resilience.ExponentialBackoff(100*time.Millisecond, 2.0, 5*time.Second),
//	    ShouldRetry: resilience.DefaultShouldRetry,
//	}
//
//	err := resilience.Retry(ctx, config, func(ctx context.Context) error {
//	    return performOperation()
//	})
//
// Circuit Breaker Pattern:
//
//	config := &resilience.CircuitBreakerConfig{
//	    MaxFailures: 5,
//	    Timeout:     60 * time.Second,
//	    MaxHalfOpenRequests: 1,
//	}
//	cb := resilience.NewCircuitBreaker(config)
//
//	err := cb.Execute(ctx, func(ctx context.Context) error {
//	    return performOperation()
//	})
//
// Bulkhead Pattern:
//
//	config := &resilience.BulkheadConfig{
//	    MaxConcurrent: 10,
//	    MaxQueueDepth: 100,
//	    Timeout:       5 * time.Second,
//	}
//	bulkhead := resilience.NewBulkhead(config)
//
//	err := bulkhead.Execute(ctx, func(ctx context.Context) error {
//	    return performOperation()
//	})
//
// Timeout Pattern:
//
//	config := &resilience.TimeoutConfig{
//	    Duration: 30 * time.Second,
//	}
//
//	err := resilience.WithTimeout(ctx, config, func(ctx context.Context) error {
//	    return performOperation()
//	})
//
// Combining Patterns:
//
//	// Retry with circuit breaker
//	cb := resilience.NewCircuitBreaker(nil)
//	retryConfig := resilience.DefaultRetryConfig()
//
//	err := resilience.Retry(ctx, retryConfig, func(ctx context.Context) error {
//	    return cb.Execute(ctx, func(ctx context.Context) error {
//	        return performOperation()
//	    })
//	})
//
//	// Circuit breaker with timeout and bulkhead
//	cb := resilience.NewCircuitBreaker(nil)
//	bulkhead := resilience.NewBulkhead(nil)
//	timeoutConfig := &resilience.TimeoutConfig{Duration: 10 * time.Second}
//
//	err := cb.Execute(ctx, func(ctx context.Context) error {
//	    return bulkhead.Execute(ctx, func(ctx context.Context) error {
//	        return resilience.WithTimeout(ctx, timeoutConfig, func(ctx context.Context) error {
//	            return performOperation()
//	        })
//	    })
//	})
package resilience
