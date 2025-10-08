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

package resilience

import (
	"context"
	"time"
)

// Executor is a function that performs an operation that may fail.
type Executor func(ctx context.Context) error

// ShouldRetry determines if an error is retryable.
type ShouldRetry func(err error) bool

// BackoffStrategy determines the delay between retries.
type BackoffStrategy func(attempt int) time.Duration

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	MaxAttempts int

	// Backoff is the backoff strategy.
	Backoff BackoffStrategy

	// ShouldRetry determines if an error should trigger a retry.
	ShouldRetry ShouldRetry

	// OnRetry is called before each retry attempt.
	OnRetry func(attempt int, err error)
}

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	// MaxFailures is the maximum number of consecutive failures before opening.
	MaxFailures int

	// Timeout is how long to wait in Open state before trying Half-Open.
	Timeout time.Duration

	// MaxHalfOpenRequests is the maximum number of requests allowed in Half-Open state.
	MaxHalfOpenRequests int

	// OnStateChange is called when the circuit breaker changes state.
	OnStateChange func(from, to State)
}

// State represents the circuit breaker state.
type State int

const (
	// StateClosed means the circuit is closed (normal operation).
	StateClosed State = iota

	// StateOpen means the circuit is open (failing fast).
	StateOpen

	// StateHalfOpen means the circuit is half-open (testing recovery).
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// BulkheadConfig configures bulkhead isolation.
type BulkheadConfig struct {
	// MaxConcurrent is the maximum number of concurrent executions.
	MaxConcurrent int

	// MaxQueueDepth is the maximum number of queued executions (0 = no queue).
	MaxQueueDepth int

	// Timeout is the maximum time to wait for a slot.
	Timeout time.Duration
}

// TimeoutConfig configures timeout behavior.
type TimeoutConfig struct {
	// Duration is the timeout duration.
	Duration time.Duration
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		Backoff:     ExponentialBackoff(100*time.Millisecond, 2.0, 5*time.Second),
		ShouldRetry: DefaultShouldRetry,
		OnRetry:     nil,
	}
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             60 * time.Second,
		MaxHalfOpenRequests: 1,
		OnStateChange:       nil,
	}
}

// DefaultBulkheadConfig returns a default bulkhead configuration.
func DefaultBulkheadConfig() *BulkheadConfig {
	return &BulkheadConfig{
		MaxConcurrent: 10,
		MaxQueueDepth: 0,
		Timeout:       5 * time.Second,
	}
}

// DefaultTimeoutConfig returns a default timeout configuration.
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Duration: 30 * time.Second,
	}
}
