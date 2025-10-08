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
	"time"
)

func TestNewStartupChecker(t *testing.T) {
	checker := NewStartupChecker()
	if checker == nil {
		t.Fatal("NewStartupChecker() returned nil")
	}
	if checker.ready {
		t.Error("expected new startup checker to not be ready")
	}
	if checker.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
}

func TestStartupChecker_Name(t *testing.T) {
	checker := NewStartupChecker()
	if got := checker.Name(); got != "startup" {
		t.Errorf("Name() = %v, want %v", got, "startup")
	}
}

func TestStartupChecker_Check_NotReady(t *testing.T) {
	checker := NewStartupChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Name != "startup" {
		t.Errorf("result.Name = %v, want %v", result.Name, "startup")
	}
	if result.Status != StatusUnhealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusUnhealthy)
	}
	if result.Message != "startup in progress" {
		t.Errorf("result.Message = %v, want %v", result.Message, "startup in progress")
	}
	if result.Details == nil {
		t.Error("expected Details to be set")
	}
	if _, ok := result.Details["elapsed_ms"]; !ok {
		t.Error("expected elapsed_ms in Details")
	}
}

func TestStartupChecker_Check_Ready(t *testing.T) {
	checker := NewStartupChecker()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure measurable duration
	checker.MarkReady()
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Name != "startup" {
		t.Errorf("result.Name = %v, want %v", result.Name, "startup")
	}
	if result.Status != StatusHealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusHealthy)
	}
	if result.Message != "startup completed" {
		t.Errorf("result.Message = %v, want %v", result.Message, "startup completed")
	}
	if result.Details == nil {
		t.Error("expected Details to be set")
	}
	duration, ok := result.Details["startup_duration_ms"]
	if !ok {
		t.Error("expected startup_duration_ms in Details")
	}
	if durationInt, ok := duration.(int64); !ok || durationInt <= 0 {
		t.Errorf("expected positive startup_duration_ms, got %v", duration)
	}
}

func TestStartupChecker_MarkReady(t *testing.T) {
	checker := NewStartupChecker()

	// Verify not ready initially
	if checker.IsReady() {
		t.Error("expected checker to not be ready initially")
	}

	// Mark ready
	checker.MarkReady()

	// Verify ready
	if !checker.IsReady() {
		t.Error("expected checker to be ready after MarkReady")
	}

	// Verify Check returns healthy
	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Error("expected healthy status after MarkReady")
	}
}

func TestStartupChecker_MarkReady_Idempotent(t *testing.T) {
	checker := NewStartupChecker()
	time.Sleep(10 * time.Millisecond)
	checker.MarkReady()

	// Get first ready time
	firstResult := checker.Check(context.Background())
	firstDuration := firstResult.Details["startup_duration_ms"]

	// Mark ready again after delay
	time.Sleep(10 * time.Millisecond)
	checker.MarkReady()

	// Get second ready time
	secondResult := checker.Check(context.Background())
	secondDuration := secondResult.Details["startup_duration_ms"]

	// Duration should be the same (first MarkReady wins)
	if firstDuration != secondDuration {
		t.Errorf("expected duration to remain %v, got %v", firstDuration, secondDuration)
	}
}

func TestStartupChecker_IsReady(t *testing.T) {
	checker := NewStartupChecker()

	if checker.IsReady() {
		t.Error("expected IsReady() = false initially")
	}

	checker.MarkReady()

	if !checker.IsReady() {
		t.Error("expected IsReady() = true after MarkReady")
	}
}

func TestStartupChecker_Reset(t *testing.T) {
	checker := NewStartupChecker()
	checker.MarkReady()

	// Verify ready
	if !checker.IsReady() {
		t.Error("expected checker to be ready before reset")
	}

	// Reset
	oldStartTime := checker.startTime
	time.Sleep(10 * time.Millisecond)
	checker.Reset()

	// Verify not ready after reset
	if checker.IsReady() {
		t.Error("expected checker to not be ready after reset")
	}

	// Verify start time was updated
	if !checker.startTime.After(oldStartTime) {
		t.Error("expected startTime to be updated after reset")
	}

	// Verify Check returns unhealthy
	result := checker.Check(context.Background())
	if result.Status != StatusUnhealthy {
		t.Error("expected unhealthy status after reset")
	}
}

func TestStartupChecker_Concurrent(t *testing.T) {
	checker := NewStartupChecker()
	ctx := context.Background()

	// Run concurrent checks and state changes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				checker.Check(ctx)
				if id%2 == 0 {
					checker.MarkReady()
				} else {
					_ = checker.IsReady()
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
}
