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

// mockChecker is a mock implementation of Checker for testing
type mockChecker struct {
	name   string
	result CheckResult
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context) CheckResult {
	return m.result
}

func TestNewReadinessChecker(t *testing.T) {
	checker := NewReadinessChecker()
	if checker == nil {
		t.Fatal("NewReadinessChecker() returned nil")
	}
	if len(checker.checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(checker.checks))
	}
}

func TestNewReadinessChecker_WithChecks(t *testing.T) {
	mock1 := &mockChecker{name: "check1"}
	mock2 := &mockChecker{name: "check2"}

	checker := NewReadinessChecker(mock1, mock2)
	if len(checker.checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(checker.checks))
	}
}

func TestReadinessChecker_Name(t *testing.T) {
	checker := NewReadinessChecker()
	if got := checker.Name(); got != "readiness" {
		t.Errorf("Name() = %v, want %v", got, "readiness")
	}
}

func TestReadinessChecker_Check_AllHealthy(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusHealthy},
	}

	checker := NewReadinessChecker(mock1, mock2)
	result := checker.Check(context.Background())

	if result.Name != "readiness" {
		t.Errorf("result.Name = %v, want %v", result.Name, "readiness")
	}
	if result.Status != StatusHealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusHealthy)
	}
	if result.Message != "" {
		t.Errorf("result.Message = %v, want empty", result.Message)
	}
	if result.Details == nil {
		t.Fatal("expected Details to be set")
	}
	checks, ok := result.Details["checks"].([]CheckResult)
	if !ok {
		t.Fatal("expected checks in Details")
	}
	if len(checks) != 2 {
		t.Errorf("expected 2 checks in details, got %d", len(checks))
	}
}

func TestReadinessChecker_Check_OneDegraded(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusDegraded, Message: "slow"},
	}

	checker := NewReadinessChecker(mock1, mock2)
	result := checker.Check(context.Background())

	if result.Name != "readiness" {
		t.Errorf("result.Name = %v, want %v", result.Name, "readiness")
	}
	if result.Status != StatusDegraded {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusDegraded)
	}
	if result.Message != "one or more dependencies degraded" {
		t.Errorf("result.Message = %v, want 'one or more dependencies degraded'", result.Message)
	}
}

func TestReadinessChecker_Check_OneUnhealthy(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusUnhealthy, Message: "down"},
	}

	checker := NewReadinessChecker(mock1, mock2)
	result := checker.Check(context.Background())

	if result.Name != "readiness" {
		t.Errorf("result.Name = %v, want %v", result.Name, "readiness")
	}
	if result.Status != StatusUnhealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusUnhealthy)
	}
	if result.Message != "one or more dependencies unhealthy" {
		t.Errorf("result.Message = %v, want 'one or more dependencies unhealthy'", result.Message)
	}
}

func TestReadinessChecker_Check_UnhealthyOverridesDegraded(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusDegraded, Message: "slow"},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusUnhealthy, Message: "down"},
	}

	checker := NewReadinessChecker(mock1, mock2)
	result := checker.Check(context.Background())

	if result.Status != StatusUnhealthy {
		t.Errorf("result.Status = %v, want %v (unhealthy should override degraded)", result.Status, StatusUnhealthy)
	}
}

func TestReadinessChecker_Check_Empty(t *testing.T) {
	checker := NewReadinessChecker()
	result := checker.Check(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("result.Status = %v, want %v (empty checks should be healthy)", result.Status, StatusHealthy)
	}
}

func TestReadinessChecker_AddCheck(t *testing.T) {
	checker := NewReadinessChecker()

	// Initially empty
	if len(checker.checks) != 0 {
		t.Errorf("expected 0 checks initially, got %d", len(checker.checks))
	}

	// Add check
	mock := &mockChecker{name: "test"}
	checker.AddCheck(mock)

	if len(checker.checks) != 1 {
		t.Errorf("expected 1 check after add, got %d", len(checker.checks))
	}

	// Add another
	mock2 := &mockChecker{name: "test2"}
	checker.AddCheck(mock2)

	if len(checker.checks) != 2 {
		t.Errorf("expected 2 checks after second add, got %d", len(checker.checks))
	}
}

func TestReadinessChecker_RemoveCheck(t *testing.T) {
	mock1 := &mockChecker{name: "check1"}
	mock2 := &mockChecker{name: "check2"}
	mock3 := &mockChecker{name: "check3"}

	checker := NewReadinessChecker(mock1, mock2, mock3)

	// Remove middle check
	checker.RemoveCheck("check2")

	if len(checker.checks) != 2 {
		t.Errorf("expected 2 checks after remove, got %d", len(checker.checks))
	}

	// Verify remaining checks
	found1, found2, found3 := false, false, false
	for _, check := range checker.checks {
		switch check.Name() {
		case "check1":
			found1 = true
		case "check2":
			found2 = true
		case "check3":
			found3 = true
		}
	}

	if !found1 || found2 || !found3 {
		t.Error("expected check1 and check3 to remain, check2 to be removed")
	}
}

func TestReadinessChecker_RemoveCheck_NotFound(t *testing.T) {
	mock1 := &mockChecker{name: "check1"}
	checker := NewReadinessChecker(mock1)

	// Remove non-existent check
	checker.RemoveCheck("nonexistent")

	// Should still have 1 check
	if len(checker.checks) != 1 {
		t.Errorf("expected 1 check after removing non-existent, got %d", len(checker.checks))
	}
}

func TestReadinessChecker_Concurrent(t *testing.T) {
	mock := &mockChecker{
		name:   "test",
		result: CheckResult{Name: "test", Status: StatusHealthy},
	}
	checker := NewReadinessChecker(mock)
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				if id%3 == 0 {
					checker.Check(ctx)
				} else if id%3 == 1 {
					checker.AddCheck(&mockChecker{name: "temp"})
				} else {
					checker.RemoveCheck("temp")
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
