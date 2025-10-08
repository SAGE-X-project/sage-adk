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
	"sync"
	"time"
)

// StartupChecker checks if the agent has completed startup.
type StartupChecker struct {
	ready     bool
	startTime time.Time
	readyTime *time.Time
	mu        sync.RWMutex
}

// NewStartupChecker creates a new startup checker.
func NewStartupChecker() *StartupChecker {
	return &StartupChecker{
		ready:     false,
		startTime: time.Now(),
	}
}

// Name returns the name of this health check.
func (c *StartupChecker) Name() string {
	return "startup"
}

// Check performs the startup check.
func (c *StartupChecker) Check(ctx context.Context) CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ready {
		details := map[string]interface{}{
			"startup_duration_ms": c.getStartupDuration(),
		}

		return CheckResult{
			Name:    c.Name(),
			Status:  StatusHealthy,
			Message: "startup completed",
			Details: details,
		}
	}

	elapsed := time.Since(c.startTime)
	return CheckResult{
		Name:    c.Name(),
		Status:  StatusUnhealthy,
		Message: "startup in progress",
		Details: map[string]interface{}{
			"elapsed_ms": elapsed.Milliseconds(),
		},
	}
}

// MarkReady marks the startup as complete.
func (c *StartupChecker) MarkReady() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.ready {
		now := time.Now()
		c.readyTime = &now
		c.ready = true
	}
}

// IsReady returns true if startup is complete.
func (c *StartupChecker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// getStartupDuration returns the startup duration in milliseconds.
func (c *StartupChecker) getStartupDuration() int64 {
	if c.readyTime == nil {
		return 0
	}
	return c.readyTime.Sub(c.startTime).Milliseconds()
}

// Reset resets the startup checker (useful for testing).
func (c *StartupChecker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ready = false
	c.startTime = time.Now()
	c.readyTime = nil
}
