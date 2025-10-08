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

import "testing"

func TestCheckResult_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "healthy status",
			status: StatusHealthy,
			want:   true,
		},
		{
			name:   "unhealthy status",
			status: StatusUnhealthy,
			want:   false,
		},
		{
			name:   "degraded status",
			status: StatusDegraded,
			want:   false,
		},
		{
			name:   "unknown status",
			status: StatusUnknown,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckResult{Status: tt.status}
			if got := r.IsHealthy(); got != tt.want {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckResult_IsDegraded(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "healthy status",
			status: StatusHealthy,
			want:   false,
		},
		{
			name:   "unhealthy status",
			status: StatusUnhealthy,
			want:   false,
		},
		{
			name:   "degraded status",
			status: StatusDegraded,
			want:   true,
		},
		{
			name:   "unknown status",
			status: StatusUnknown,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckResult{Status: tt.status}
			if got := r.IsDegraded(); got != tt.want {
				t.Errorf("IsDegraded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckResult_IsUnhealthy(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "healthy status",
			status: StatusHealthy,
			want:   false,
		},
		{
			name:   "unhealthy status",
			status: StatusUnhealthy,
			want:   true,
		},
		{
			name:   "degraded status",
			status: StatusDegraded,
			want:   false,
		},
		{
			name:   "unknown status",
			status: StatusUnknown,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckResult{Status: tt.status}
			if got := r.IsUnhealthy(); got != tt.want {
				t.Errorf("IsUnhealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}
