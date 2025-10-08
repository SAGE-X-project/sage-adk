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

import "errors"

var (
	// ErrCircuitBreakerOpen is returned when the circuit breaker is open.
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// ErrMaxAttemptsExceeded is returned when retry attempts are exhausted.
	ErrMaxAttemptsExceeded = errors.New("maximum retry attempts exceeded")

	// ErrBulkheadFull is returned when the bulkhead is at capacity.
	ErrBulkheadFull = errors.New("bulkhead is full")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")
)
