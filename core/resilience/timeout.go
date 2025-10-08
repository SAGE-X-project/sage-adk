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
)

// WithTimeout executes the function with a timeout.
func WithTimeout(ctx context.Context, config *TimeoutConfig, fn Executor) error {
	if config == nil {
		config = DefaultTimeoutConfig()
	}

	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Channel to receive result
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Execute function in goroutine
	go func() {
		resultChan <- result{err: fn(ctx)}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		return res.err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimeout
		}
		return ctx.Err()
	}
}
