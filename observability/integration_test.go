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

package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sage-x-project/sage-adk/observability/health"
)

func TestNewManager(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config: &Config{
			Metrics: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
			Logging: LoggingConfig{
				Level:        "info",
				Format:       "json",
				SamplingRate: 1.0,
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	// Verify components
	if manager.Logger() == nil {
		t.Error("expected non-nil logger")
	}
	if manager.Collector() == nil {
		t.Error("expected non-nil collector")
	}
	if manager.AgentMetrics() == nil {
		t.Error("expected non-nil agent metrics")
	}
	if manager.LLMMetrics() == nil {
		t.Error("expected non-nil LLM metrics")
	}
	if manager.Middleware() == nil {
		t.Error("expected non-nil middleware")
	}
	if manager.LivenessChecker() == nil {
		t.Error("expected non-nil liveness checker")
	}
	if manager.StartupChecker() == nil {
		t.Error("expected non-nil startup checker")
	}
	if manager.ReadinessChecker() == nil {
		t.Error("expected non-nil readiness checker")
	}
}

func TestNewManager_InvalidConfig(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config: &Config{
			Metrics: MetricsConfig{
				Enabled: true,
				Port:    -1, // Invalid port
			},
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestManager_MarkReady(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config:  DefaultConfig(),
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Initially not ready
	if manager.StartupChecker().IsReady() {
		t.Error("expected startup checker to not be ready initially")
	}

	// Mark ready
	manager.MarkReady()

	// Verify ready
	if !manager.StartupChecker().IsReady() {
		t.Error("expected startup checker to be ready after MarkReady")
	}
}

func TestManager_AddReadinessCheck(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config:  DefaultConfig(),
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Add custom health check
	customCheck := &mockHealthChecker{
		name:   "custom",
		status: health.StatusHealthy,
	}
	manager.AddReadinessCheck(customCheck)

	// Mark startup ready so readiness check passes
	manager.MarkReady()

	// Verify check was added
	result := manager.ReadinessChecker().Check(context.Background())
	if result.Status != health.StatusHealthy {
		t.Errorf("expected healthy status, got %v", result.Status)
	}
}

func TestManager_HTTPHandler(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config:  DefaultConfig(),
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	handler := manager.HTTPHandler()
	if handler == nil {
		t.Fatal("expected non-nil HTTP handler")
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "metrics endpoint",
			path:       "/metrics",
			wantStatus: http.StatusOK,
		},
		{
			name:       "liveness endpoint",
			path:       "/health/live",
			wantStatus: http.StatusOK,
		},
		{
			name:       "readiness endpoint - not ready",
			path:       "/health/ready",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "startup endpoint - not ready",
			path:       "/health/startup",
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestManager_HTTPHandler_AfterReady(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config:  DefaultConfig(),
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Mark ready
	manager.MarkReady()

	handler := manager.HTTPHandler()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "readiness endpoint - ready",
			path:       "/health/ready",
			wantStatus: http.StatusOK,
		},
		{
			name:       "startup endpoint - ready",
			path:       "/health/startup",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestManager_Shutdown(t *testing.T) {
	cfg := &ManagerConfig{
		AgentID: "test-agent",
		Config:  DefaultConfig(),
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Verify liveness is healthy before shutdown
	result := manager.LivenessChecker().Check(context.Background())
	if result.Status != health.StatusHealthy {
		t.Error("expected healthy status before shutdown")
	}

	// Shutdown
	ctx := context.Background()
	if err := manager.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Verify liveness is unhealthy after shutdown
	result = manager.LivenessChecker().Check(context.Background())
	if result.Status != health.StatusUnhealthy {
		t.Error("expected unhealthy status after shutdown")
	}
}

// mockHealthChecker is a mock health checker for testing
type mockHealthChecker struct {
	name   string
	status health.Status
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check(ctx context.Context) health.CheckResult {
	return health.CheckResult{
		Name:   m.name,
		Status: m.status,
	}
}
