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
	"fmt"
	"math"
	"time"
)

// Retry executes the function with retry logic.
func Retry(ctx context.Context, config *RetryConfig, fn Executor) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context
		if err := ctx.Err(); err != nil {
			return err
		}

		// Execute function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !config.ShouldRetry(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Last attempt - don't wait
		if attempt == config.MaxAttempts {
			break
		}

		// Call OnRetry callback
		if config.OnRetry != nil {
			config.OnRetry(attempt, err)
		}

		// Calculate backoff delay
		delay := config.Backoff(attempt)

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("%w: last error: %v", ErrMaxAttemptsExceeded, lastErr)
}

// ConstantBackoff creates a backoff strategy with constant delay.
func ConstantBackoff(delay time.Duration) BackoffStrategy {
	return func(attempt int) time.Duration {
		return delay
	}
}

// LinearBackoff creates a backoff strategy with linear increase.
func LinearBackoff(base time.Duration, max time.Duration) BackoffStrategy {
	return func(attempt int) time.Duration {
		delay := base * time.Duration(attempt)
		if delay > max {
			delay = max
		}
		return delay
	}
}

// ExponentialBackoff creates a backoff strategy with exponential increase.
func ExponentialBackoff(base time.Duration, multiplier float64, max time.Duration) BackoffStrategy {
	return func(attempt int) time.Duration {
		delay := float64(base) * math.Pow(multiplier, float64(attempt-1))
		duration := time.Duration(delay)
		if duration > max {
			duration = max
		}
		return duration
	}
}

// DefaultShouldRetry is a default retry predicate that retries on any error.
func DefaultShouldRetry(err error) bool {
	return err != nil
}

// NeverRetry never retries.
func NeverRetry(err error) bool {
	return false
}

// RetryOnSpecificErrors creates a retry predicate that only retries specific errors.
func RetryOnSpecificErrors(errors ...error) ShouldRetry {
	errorMap := make(map[error]bool)
	for _, err := range errors {
		errorMap[err] = true
	}

	return func(err error) bool {
		return errorMap[err]
	}
}
