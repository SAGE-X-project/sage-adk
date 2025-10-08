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
)

// LivenessChecker checks if the agent is alive.
type LivenessChecker struct {
	running bool
	mu      sync.RWMutex
}

// NewLivenessChecker creates a new liveness checker.
func NewLivenessChecker() *LivenessChecker {
	return &LivenessChecker{
		running: true,
	}
}

// Name returns the name of this health check.
func (c *LivenessChecker) Name() string {
	return "liveness"
}

// Check performs the liveness check.
func (c *LivenessChecker) Check(ctx context.Context) CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.running {
		return CheckResult{
			Name:   c.Name(),
			Status: StatusHealthy,
		}
	}

	return CheckResult{
		Name:    c.Name(),
		Status:  StatusUnhealthy,
		Message: "agent not running",
	}
}

// MarkRunning marks the agent as running.
func (c *LivenessChecker) MarkRunning() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = true
}

// MarkStopped marks the agent as stopped.
func (c *LivenessChecker) MarkStopped() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
}
