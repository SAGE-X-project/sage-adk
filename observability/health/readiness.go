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

// ReadinessChecker checks if the agent is ready to serve traffic.
type ReadinessChecker struct {
	checks []Checker
	mu     sync.RWMutex
}

// NewReadinessChecker creates a new readiness checker.
func NewReadinessChecker(checks ...Checker) *ReadinessChecker {
	return &ReadinessChecker{
		checks: checks,
	}
}

// Name returns the name of this health check.
func (c *ReadinessChecker) Name() string {
	return "readiness"
}

// Check performs the readiness check.
func (c *ReadinessChecker) Check(ctx context.Context) CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make([]CheckResult, 0, len(c.checks))

	for _, check := range c.checks {
		result := check.Check(ctx)
		results = append(results, result)

		// If any check is unhealthy, the agent is not ready
		if result.IsUnhealthy() {
			return CheckResult{
				Name:    c.Name(),
				Status:  StatusUnhealthy,
				Message: "one or more dependencies unhealthy",
				Details: map[string]interface{}{
					"checks": results,
				},
			}
		}
	}

	// Check for degraded status
	degraded := false
	for _, result := range results {
		if result.IsDegraded() {
			degraded = true
			break
		}
	}

	status := StatusHealthy
	message := ""
	if degraded {
		status = StatusDegraded
		message = "one or more dependencies degraded"
	}

	return CheckResult{
		Name:    c.Name(),
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"checks": results,
		},
	}
}

// AddCheck adds a health check to the readiness checker.
func (c *ReadinessChecker) AddCheck(check Checker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check)
}

// RemoveCheck removes a health check by name.
func (c *ReadinessChecker) RemoveCheck(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filtered := make([]Checker, 0, len(c.checks))
	for _, check := range c.checks {
		if check.Name() != name {
			filtered = append(filtered, check)
		}
	}
	c.checks = filtered
}
