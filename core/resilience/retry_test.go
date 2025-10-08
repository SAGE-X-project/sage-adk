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

func TestRetry_Success(t *testing.T) {
	attempts := 0
	config := &RetryConfig{
		MaxAttempts: 3,
		Backoff:     ConstantBackoff(10 * time.Millisecond),
		ShouldRetry: DefaultShouldRetry,
	}

	err := Retry(context.Background(), config, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRetry_MaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	config := &RetryConfig{
		MaxAttempts: 3,
		Backoff:     ConstantBackoff(1 * time.Millisecond),
		ShouldRetry: DefaultShouldRetry,
	}

	err := Retry(context.Background(), config, func(ctx context.Context) error {
		attempts++
		return errors.New("persistent error")
	})

	if !errors.Is(err, ErrMaxAttemptsExceeded) {
		t.Errorf("Retry() error = %v, want ErrMaxAttemptsExceeded", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	attempts := 0
	nonRetryableErr := errors.New("non-retryable error")

	config := &RetryConfig{
		MaxAttempts: 3,
		Backoff:     ConstantBackoff(1 * time.Millisecond),
		ShouldRetry: func(err error) bool {
			return err != nonRetryableErr
		},
	}

	err := Retry(context.Background(), config, func(ctx context.Context) error {
		attempts++
		return nonRetryableErr
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry)", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	config := &RetryConfig{
		MaxAttempts: 10,
		Backoff:     ConstantBackoff(50 * time.Millisecond),
		ShouldRetry: DefaultShouldRetry,
	}

	// Cancel after first failure
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, config, func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	})

	if err != context.Canceled {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}
	if attempts > 2 {
		t.Errorf("attempts = %d, want <= 2 (context should cancel)", attempts)
	}
}

func TestRetry_OnRetryCallback(t *testing.T) {
	var retryAttempts []int
	config := &RetryConfig{
		MaxAttempts: 3,
		Backoff:     ConstantBackoff(1 * time.Millisecond),
		ShouldRetry: DefaultShouldRetry,
		OnRetry: func(attempt int, err error) {
			retryAttempts = append(retryAttempts, attempt)
		},
	}

	Retry(context.Background(), config, func(ctx context.Context) error {
		return errors.New("error")
	})

	if len(retryAttempts) != 2 {
		t.Errorf("retry callbacks = %d, want 2", len(retryAttempts))
	}
}

func TestConstantBackoff(t *testing.T) {
	backoff := ConstantBackoff(100 * time.Millisecond)

	for i := 1; i <= 5; i++ {
		delay := backoff(i)
		if delay != 100*time.Millisecond {
			t.Errorf("delay = %v, want 100ms", delay)
		}
	}
}

func TestLinearBackoff(t *testing.T) {
	backoff := LinearBackoff(100*time.Millisecond, 500*time.Millisecond)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
		{5, 500 * time.Millisecond}, // capped at max
		{10, 500 * time.Millisecond}, // capped at max
	}

	for _, tt := range tests {
		delay := backoff(tt.attempt)
		if delay != tt.want {
			t.Errorf("backoff(%d) = %v, want %v", tt.attempt, delay, tt.want)
		}
	}
}

func TestExponentialBackoff(t *testing.T) {
	backoff := ExponentialBackoff(100*time.Millisecond, 2.0, 1*time.Second)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1 * time.Second}, // capped at max
	}

	for _, tt := range tests {
		delay := backoff(tt.attempt)
		if delay != tt.want {
			t.Errorf("backoff(%d) = %v, want %v", tt.attempt, delay, tt.want)
		}
	}
}

func TestDefaultShouldRetry(t *testing.T) {
	if !DefaultShouldRetry(errors.New("error")) {
		t.Error("DefaultShouldRetry should return true for errors")
	}
	if DefaultShouldRetry(nil) {
		t.Error("DefaultShouldRetry should return false for nil")
	}
}

func TestNeverRetry(t *testing.T) {
	if NeverRetry(errors.New("error")) {
		t.Error("NeverRetry should return false")
	}
}

func TestRetryOnSpecificErrors(t *testing.T) {
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	err3 := errors.New("error3")

	shouldRetry := RetryOnSpecificErrors(err1, err2)

	if !shouldRetry(err1) {
		t.Error("should retry error1")
	}
	if !shouldRetry(err2) {
		t.Error("should retry error2")
	}
	if shouldRetry(err3) {
		t.Error("should not retry error3")
	}
}

func TestRetry_DefaultConfig(t *testing.T) {
	attempts := 0

	err := Retry(context.Background(), nil, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
}
