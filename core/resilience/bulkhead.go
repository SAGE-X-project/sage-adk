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

// Bulkhead implements the bulkhead isolation pattern.
type Bulkhead struct {
	config     *BulkheadConfig
	semaphore  chan struct{}
	queueDepth chan struct{}
}

// NewBulkhead creates a new bulkhead.
func NewBulkhead(config *BulkheadConfig) *Bulkhead {
	if config == nil {
		config = DefaultBulkheadConfig()
	}

	b := &Bulkhead{
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}

	if config.MaxQueueDepth > 0 {
		b.queueDepth = make(chan struct{}, config.MaxQueueDepth)
	}

	return b
}

// Execute executes the function with bulkhead isolation.
func (b *Bulkhead) Execute(ctx context.Context, fn Executor) error {
	// Try to enter queue if configured
	if b.queueDepth != nil {
		select {
		case b.queueDepth <- struct{}{}:
			defer func() { <-b.queueDepth }()
		default:
			return ErrBulkheadFull
		}
	}

	// Try to acquire semaphore
	if b.config.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, b.config.Timeout)
		defer cancel()

		select {
		case b.semaphore <- struct{}{}:
			defer func() { <-b.semaphore }()
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return ErrBulkheadFull
			}
			return ctx.Err()
		}
	} else {
		select {
		case b.semaphore <- struct{}{}:
			defer func() { <-b.semaphore }()
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Execute function
	return fn(ctx)
}

// Available returns the number of available slots.
func (b *Bulkhead) Available() int {
	return b.config.MaxConcurrent - len(b.semaphore)
}

// InProgress returns the number of in-progress executions.
func (b *Bulkhead) InProgress() int {
	return len(b.semaphore)
}

// QueueLength returns the current queue length.
func (b *Bulkhead) QueueLength() int {
	if b.queueDepth == nil {
		return 0
	}
	return len(b.queueDepth)
}
