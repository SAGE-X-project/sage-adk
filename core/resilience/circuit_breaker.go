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
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu                  sync.RWMutex
	config              *CircuitBreakerConfig
	state               State
	failures            int
	halfOpenRequests    int
	lastStateChangeTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config:              config,
		state:               StateClosed,
		failures:            0,
		halfOpenRequests:    0,
		lastStateChangeTime: time.Now(),
	}
}

// Execute executes the function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn Executor) error {
	// Check if we can execute
	if !cb.canExecute() {
		return ErrCircuitBreakerOpen
	}

	// Execute function
	err := fn(ctx)

	// Record result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// canExecute checks if the circuit breaker allows execution.
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateChangeTime) >= cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 0
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenRequests < cb.config.MaxHalfOpenRequests {
			cb.halfOpenRequests++
			return true
		}
		return false

	default:
		return false
	}
}

// recordSuccess records a successful execution.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		// Success in half-open state - close the circuit
		cb.setState(StateClosed)
		cb.failures = 0
		cb.halfOpenRequests = 0
	} else if cb.state == StateClosed {
		// Reset failure count on success
		cb.failures = 0
	}
}

// recordFailure records a failed execution.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++

	if cb.state == StateHalfOpen {
		// Failure in half-open state - reopen the circuit
		cb.setState(StateOpen)
		cb.halfOpenRequests = 0
	} else if cb.state == StateClosed && cb.failures >= cb.config.MaxFailures {
		// Too many failures in closed state - open the circuit
		cb.setState(StateOpen)
	}
}

// setState changes the circuit breaker state.
func (cb *CircuitBreaker) setState(newState State) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChangeTime = time.Now()

	if cb.config.OnStateChange != nil && oldState != newState {
		// Call callback outside the lock to prevent deadlock
		go cb.config.OnStateChange(oldState, newState)
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Failures returns the current failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenRequests = 0
	cb.lastStateChangeTime = time.Now()

	if cb.config.OnStateChange != nil && oldState != StateClosed {
		go cb.config.OnStateChange(oldState, StateClosed)
	}
}
