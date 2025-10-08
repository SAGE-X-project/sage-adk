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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Healthy(t *testing.T) {
	mock := &mockChecker{
		name:   "test",
		result: CheckResult{Name: "test", Status: StatusHealthy, Message: "all good"},
	}

	handler := Handler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Check response body
	var result CheckResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("result.Name = %v, want %v", result.Name, "test")
	}
	if result.Status != StatusHealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusHealthy)
	}
}

func TestHandler_Unhealthy(t *testing.T) {
	mock := &mockChecker{
		name:   "test",
		result: CheckResult{Name: "test", Status: StatusUnhealthy, Message: "something wrong"},
	}

	handler := Handler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	// Check response body
	var result CheckResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusUnhealthy)
	}
}

func TestHandler_Degraded(t *testing.T) {
	mock := &mockChecker{
		name:   "test",
		result: CheckResult{Name: "test", Status: StatusDegraded, Message: "slow"},
	}

	handler := Handler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code - degraded should return 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for degraded, got %d", http.StatusOK, rec.Code)
	}

	// Check response body
	var result CheckResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Status != StatusDegraded {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusDegraded)
	}
}

func TestMultiHandler_AllHealthy(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusHealthy},
	}

	handler := MultiHandler(mock1, mock2)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := response["status"]
	if !ok {
		t.Fatal("expected status in response")
	}
	if status != string(StatusHealthy) {
		t.Errorf("response status = %v, want %v", status, StatusHealthy)
	}

	checks, ok := response["checks"]
	if !ok {
		t.Fatal("expected checks in response")
	}
	checksSlice, ok := checks.([]interface{})
	if !ok || len(checksSlice) != 2 {
		t.Errorf("expected 2 checks in response, got %v", checks)
	}
}

func TestMultiHandler_OneUnhealthy(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusUnhealthy, Message: "down"},
	}

	handler := MultiHandler(mock1, mock2)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := response["status"]
	if !ok {
		t.Fatal("expected status in response")
	}
	if status != string(StatusUnhealthy) {
		t.Errorf("response status = %v, want %v", status, StatusUnhealthy)
	}
}

func TestMultiHandler_OneDegraded(t *testing.T) {
	mock1 := &mockChecker{
		name:   "check1",
		result: CheckResult{Name: "check1", Status: StatusHealthy},
	}
	mock2 := &mockChecker{
		name:   "check2",
		result: CheckResult{Name: "check2", Status: StatusDegraded, Message: "slow"},
	}

	handler := MultiHandler(mock1, mock2)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code - degraded should return 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for degraded, got %d", http.StatusOK, rec.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := response["status"]
	if !ok {
		t.Fatal("expected status in response")
	}
	if status != string(StatusDegraded) {
		t.Errorf("response status = %v, want %v", status, StatusDegraded)
	}
}

func TestMultiHandler_Empty(t *testing.T) {
	handler := MultiHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Empty checks should be healthy
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for empty checks, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := response["status"]
	if !ok {
		t.Fatal("expected status in response")
	}
	if status != string(StatusHealthy) {
		t.Errorf("response status = %v, want %v", status, StatusHealthy)
	}
}

func TestHandler_WithContext(t *testing.T) {
	mock := &mockChecker{
		name: "test",
		result: CheckResult{
			Name:   "test",
			Status: StatusHealthy,
		},
	}

	handler := Handler(mock)

	// Create request with context
	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandler_ContextCancellation(t *testing.T) {
	// Create a checker that checks context
	slowChecker := &mockChecker{
		name:   "slow",
		result: CheckResult{Name: "slow", Status: StatusHealthy},
	}

	handler := Handler(slowChecker)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Handler should still complete despite cancelled context
	// (the timeout mechanism will handle it)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
