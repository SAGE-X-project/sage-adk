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

import "context"

// Status represents the health status.
type Status string

const (
	// StatusHealthy indicates the component is healthy
	StatusHealthy Status = "healthy"

	// StatusUnhealthy indicates the component is unhealthy
	StatusUnhealthy Status = "unhealthy"

	// StatusDegraded indicates the component is degraded but functional
	StatusDegraded Status = "degraded"

	// StatusUnknown indicates the health status is unknown
	StatusUnknown Status = "unknown"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	// Name of the check
	Name string `json:"name"`

	// Status of the check
	Status Status `json:"status"`

	// Message provides additional information
	Message string `json:"message,omitempty"`

	// Details provides structured information
	Details map[string]interface{} `json:"details,omitempty"`
}

// Checker is the interface for health checks.
type Checker interface {
	// Name returns the name of this health check
	Name() string

	// Check performs the health check
	Check(ctx context.Context) CheckResult
}

// IsHealthy returns true if the status is healthy.
func (r CheckResult) IsHealthy() bool {
	return r.Status == StatusHealthy
}

// IsDegraded returns true if the status is degraded.
func (r CheckResult) IsDegraded() bool {
	return r.Status == StatusDegraded
}

// IsUnhealthy returns true if the status is unhealthy.
func (r CheckResult) IsUnhealthy() bool {
	return r.Status == StatusUnhealthy
}
