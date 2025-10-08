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
	"testing"
)

func TestNewLivenessChecker(t *testing.T) {
	checker := NewLivenessChecker()
	if checker == nil {
		t.Fatal("NewLivenessChecker() returned nil")
	}
	if !checker.running {
		t.Error("expected new liveness checker to be running")
	}
}

func TestLivenessChecker_Name(t *testing.T) {
	checker := NewLivenessChecker()
	if got := checker.Name(); got != "liveness" {
		t.Errorf("Name() = %v, want %v", got, "liveness")
	}
}

func TestLivenessChecker_Check_Running(t *testing.T) {
	checker := NewLivenessChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Name != "liveness" {
		t.Errorf("result.Name = %v, want %v", result.Name, "liveness")
	}
	if result.Status != StatusHealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusHealthy)
	}
	if result.Message != "" {
		t.Errorf("result.Message = %v, want empty", result.Message)
	}
}

func TestLivenessChecker_Check_Stopped(t *testing.T) {
	checker := NewLivenessChecker()
	checker.MarkStopped()
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Name != "liveness" {
		t.Errorf("result.Name = %v, want %v", result.Name, "liveness")
	}
	if result.Status != StatusUnhealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusUnhealthy)
	}
	if result.Message != "agent not running" {
		t.Errorf("result.Message = %v, want %v", result.Message, "agent not running")
	}
}

func TestLivenessChecker_MarkRunning(t *testing.T) {
	checker := NewLivenessChecker()
	checker.MarkStopped()

	// Verify it's stopped
	result := checker.Check(context.Background())
	if result.Status != StatusUnhealthy {
		t.Error("expected unhealthy status after MarkStopped")
	}

	// Mark running again
	checker.MarkRunning()
	result = checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Error("expected healthy status after MarkRunning")
	}
}

func TestLivenessChecker_MarkStopped(t *testing.T) {
	checker := NewLivenessChecker()

	// Verify it's running
	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Error("expected healthy status initially")
	}

	// Mark stopped
	checker.MarkStopped()
	result = checker.Check(context.Background())
	if result.Status != StatusUnhealthy {
		t.Error("expected unhealthy status after MarkStopped")
	}
}

func TestLivenessChecker_Concurrent(t *testing.T) {
	checker := NewLivenessChecker()
	ctx := context.Background()

	// Run concurrent checks and state changes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				checker.Check(ctx)
				if j%2 == 0 {
					checker.MarkRunning()
				} else {
					checker.MarkStopped()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
}
