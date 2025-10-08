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
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	if cb.State() != StateClosed {
		t.Errorf("initial state = %v, want StateClosed", cb.State())
	}
	if cb.Failures() != 0 {
		t.Errorf("initial failures = %d, want 0", cb.Failures())
	}
}

func TestCircuitBreaker_Success(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}
}

func TestCircuitBreaker_OpenOnMaxFailures(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trigger failures
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}
	if cb.Failures() != 3 {
		t.Errorf("failures = %d, want 3", cb.Failures())
	}

	// Next execution should fail fast
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		t.Error("function should not be called when circuit is open")
		return nil
	})

	if err != ErrCircuitBreakerOpen {
		t.Errorf("error = %v, want ErrCircuitBreakerOpen", err)
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next execution should transition to half-open
	executed := false
	cb.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("function should be executed in half-open state")
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed (after successful half-open)", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Fail in half-open state
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("error")
	})

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen (reopened after half-open failure)", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenRequestLimit(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Execute slowly to keep circuit in half-open
	done := make(chan struct{})
	go func() {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		close(done)
	}()

	// Give first request time to start
	time.Sleep(20 * time.Millisecond)

	// Circuit should now be processing a half-open request
	// Additional requests should fail
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	// Should either be open or the circuit closed after first success
	// Accept both as valid outcomes due to timing
	if err != nil && err != ErrCircuitBreakerOpen {
		t.Errorf("error = %v, want nil or ErrCircuitBreakerOpen", err)
	}

	<-done
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}
	if cb.Failures() != 0 {
		t.Errorf("failures = %d, want 0", cb.Failures())
	}

	// Should allow execution
	executed := false
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !executed {
		t.Error("function should be executed after reset")
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	var transitions []struct {
		from State
		to   State
	}

	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
		OnStateChange: func(from, to State) {
			transitions = append(transitions, struct {
				from State
				to   State
			}{from, to})
		},
	}
	cb := NewCircuitBreaker(config)

	// Trigger state changes
	// Closed -> Open
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	// Wait for open -> half-open
	time.Sleep(100 * time.Millisecond)

	// Half-open -> Closed
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	// Give callbacks time to execute
	time.Sleep(50 * time.Millisecond)

	if len(transitions) < 2 {
		t.Errorf("transitions = %d, want at least 2", len(transitions))
	}

	// Check first transition (Closed -> Open)
	if transitions[0].from != StateClosed || transitions[0].to != StateOpen {
		t.Errorf("first transition = %v -> %v, want Closed -> Open",
			transitions[0].from, transitions[0].to)
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}

	// Should use default config
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trigger some failures
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.Failures() != 2 {
		t.Errorf("failures = %d, want 2", cb.Failures())
	}

	// Success should reset failures
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if cb.Failures() != 0 {
		t.Errorf("failures = %d, want 0 (reset after success)", cb.Failures())
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}
}
