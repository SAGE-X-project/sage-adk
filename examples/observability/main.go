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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/observability"
	"github.com/sage-x-project/sage-adk/observability/health"
	"github.com/sage-x-project/sage-adk/observability/logging"
)

func main() {
	// Create observability configuration
	config := &observability.Config{
		Metrics: observability.MetricsConfig{
			Enabled:  true,
			Port:     9090,
			Path:     "/metrics",
			Interval: 15,
		},
		Logging: observability.LoggingConfig{
			Level:        "info",
			Format:       "json",
			Output:       "stdout",
			SamplingRate: 1.0,
		},
		Health: observability.HealthConfig{
			Enabled:       true,
			Port:          8080,
			LivenessPath:  "/health/live",
			ReadinessPath: "/health/ready",
			StartupPath:   "/health/startup",
		},
	}

	// Create observability manager
	manager, err := observability.NewManager(&observability.ManagerConfig{
		AgentID: "example-agent",
		Config:  config,
	})
	if err != nil {
		log.Fatalf("Failed to create observability manager: %v", err)
	}
	defer manager.Shutdown(context.Background())

	// Get logger
	logger := manager.Logger()
	logger.Info(context.Background(), "Starting example agent with observability")

	// Add custom readiness check
	dbChecker := &DatabaseChecker{healthy: false}
	manager.AddReadinessCheck(dbChecker)

	// Simulate database connection
	go func() {
		time.Sleep(3 * time.Second)
		dbChecker.Connect()
		logger.Info(context.Background(), "Database connected")
	}()

	// Create application server with observability middleware
	appMux := http.NewServeMux()

	// Application endpoints
	appMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Log request
		logger.Info(ctx, "Processing request", logging.String("path", r.URL.Path))

		// Record metrics
		manager.AgentMetrics().RecordRequest("example-agent", "http", 0.1)
		manager.AgentMetrics().RecordMessageReceived("example-agent", "http", "user")

		// Simulate LLM call
		manager.LLMMetrics().RecordCall("openai", "gpt-4", 0.5)
		manager.LLMMetrics().RecordTokens("openai", "gpt-4", 100, 200)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from example agent!"))
	})

	appMux.HandleFunc("/api/process", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger.Info(ctx, "Processing API request", logging.String("method", r.Method))

		// Simulate processing
		time.Sleep(50 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "processed"}`))
	})

	// Wrap application handler with observability middleware
	appHandler := manager.Middleware().Handler(appMux)

	// Start application server (port 8080)
	appServer := &http.Server{
		Addr:    ":8080",
		Handler: appHandler,
	}

	// Start observability server (metrics + health checks)
	obsServer := &http.Server{
		Addr:    ":9090",
		Handler: manager.HTTPHandler(),
	}

	// Start both servers
	go func() {
		logger.Info(context.Background(), "Starting application server on :8080")
		if err := appServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Application server error: %v", err)
		}
	}()

	go func() {
		logger.Info(context.Background(), "Starting observability server on :9090")
		if err := obsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Observability server error: %v", err)
		}
	}()

	// Mark agent as ready (after startup tasks complete)
	time.Sleep(1 * time.Second)
	manager.MarkReady()
	logger.Info(context.Background(), "Agent startup complete")

	// Print available endpoints
	fmt.Println("\n=== Example Agent with Observability ===")
	fmt.Println("Application endpoints:")
	fmt.Println("  http://localhost:8080/          - Main endpoint")
	fmt.Println("  http://localhost:8080/api/process - API endpoint")
	fmt.Println("\nObservability endpoints:")
	fmt.Println("  http://localhost:9090/metrics        - Prometheus metrics")
	fmt.Println("  http://localhost:9090/health/live    - Liveness probe")
	fmt.Println("  http://localhost:9090/health/ready   - Readiness probe")
	fmt.Println("  http://localhost:9090/health/startup - Startup probe")
	fmt.Println("\nPress Ctrl+C to stop")
	fmt.Println("========================================\n")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info(context.Background(), "Shutting down servers")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := appServer.Shutdown(ctx); err != nil {
		logger.Error(ctx, "Application server shutdown error", logging.Error(err))
	}
	if err := obsServer.Shutdown(ctx); err != nil {
		logger.Error(ctx, "Observability server shutdown error", logging.Error(err))
	}

	logger.Info(context.Background(), "Shutdown complete")
}

// DatabaseChecker is a custom health checker for database connectivity.
type DatabaseChecker struct {
	healthy bool
}

func (d *DatabaseChecker) Name() string {
	return "database"
}

func (d *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	if d.healthy {
		return health.CheckResult{
			Name:   "database",
			Status: health.StatusHealthy,
			Details: map[string]interface{}{
				"connection": "active",
			},
		}
	}

	return health.CheckResult{
		Name:    "database",
		Status:  health.StatusUnhealthy,
		Message: "database connection not established",
	}
}

func (d *DatabaseChecker) Connect() {
	d.healthy = true
}
