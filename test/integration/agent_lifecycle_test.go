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

//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/client"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// TestAgentFullLifecycle tests the complete lifecycle of an agent
func TestAgentFullLifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. Agent Creation
	t.Run("Create", func(t *testing.T) {
		agentInstance := createTestAgent(t, "lifecycle-agent")
		if agentInstance == nil {
			t.Fatal("Failed to create agent")
		}

		// Verify agent properties
		if agentInstance.Name() != "lifecycle-agent" {
			t.Errorf("Expected name 'lifecycle-agent', got '%s'", agentInstance.Name())
		}
	})

	// 2. Agent Start
	t.Run("Start", func(t *testing.T) {
		agentInstance := createTestAgent(t, "start-agent")

		// Start agent in background
		errChan := make(chan error, 1)
		go func() {
			errChan <- agentInstance.Start(":18080")
		}()

		// Wait for server to be ready
		time.Sleep(500 * time.Millisecond)

		// Check if server is running
		resp, err := http.Get("http://localhost:18080/health")
		if err != nil {
			t.Fatalf("Server not responding: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Stop agent
		if err := agentInstance.Stop(ctx); err != nil {
			t.Errorf("Failed to stop agent: %v", err)
		}
	})

	// 3. Message Processing
	t.Run("ProcessMessage", func(t *testing.T) {
		agentInstance := createTestAgent(t, "process-agent")

		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("Hello, agent!"),
		})

		response, err := agentInstance.Process(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to process message: %v", err)
		}

		if response == nil {
			t.Fatal("Response is nil")
		}

		if response.Role != types.MessageRoleAgent {
			t.Errorf("Expected role 'agent', got '%s'", response.Role)
		}
	})

	// 4. Agent Restart
	t.Run("Restart", func(t *testing.T) {
		agentInstance := createTestAgent(t, "restart-agent")

		// Start agent
		go agentInstance.Start(":18081")
		time.Sleep(500 * time.Millisecond)

		// Verify running
		resp1, err := http.Get("http://localhost:18081/health")
		if err != nil {
			t.Fatalf("Server not responding after start: %v", err)
		}
		resp1.Body.Close()

		// Stop agent
		if err := agentInstance.Stop(ctx); err != nil {
			t.Fatalf("Failed to stop agent: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Restart agent
		go agentInstance.Start(":18081")
		time.Sleep(500 * time.Millisecond)

		// Verify running again
		resp2, err := http.Get("http://localhost:18081/health")
		if err != nil {
			t.Fatalf("Server not responding after restart: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 after restart, got %d", resp2.StatusCode)
		}

		// Final cleanup
		agentInstance.Stop(ctx)
	})

	// 5. Graceful Shutdown
	t.Run("GracefulShutdown", func(t *testing.T) {
		agentInstance := createTestAgent(t, "shutdown-agent")

		// Start agent
		go agentInstance.Start(":18082")
		time.Sleep(500 * time.Millisecond)

		// Initiate graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		shutdownStart := time.Now()
		if err := agentInstance.Stop(shutdownCtx); err != nil {
			t.Errorf("Graceful shutdown failed: %v", err)
		}
		shutdownDuration := time.Since(shutdownStart)

		// Verify shutdown completed within timeout
		if shutdownDuration > 5*time.Second {
			t.Errorf("Shutdown took too long: %v", shutdownDuration)
		}

		// Verify server is no longer responding
		time.Sleep(200 * time.Millisecond)
		_, err := http.Get("http://localhost:18082/health")
		if err == nil {
			t.Error("Server still responding after shutdown")
		}
	})
}

// TestAgentStateManagement tests agent state persistence and recovery
func TestAgentStateManagement(t *testing.T) {
	ctx := context.Background()

	t.Run("StatePersistence", func(t *testing.T) {
		agentInstance := createTestAgent(t, "state-agent")

		// Process multiple messages to build state
		for i := 0; i < 5; i++ {
			msg := types.NewMessage(types.MessageRoleUser, []types.Part{
				types.NewTextPart(fmt.Sprintf("Message %d", i)),
			})

			_, err := agentInstance.Process(ctx, msg)
			if err != nil {
				t.Fatalf("Failed to process message %d: %v", i, err)
			}
		}

		// Verify state can be queried (through metrics or health endpoint)
		// This would depend on your specific state implementation
	})
}

// TestAgentHealthChecks tests Kubernetes-style health checks
func TestAgentHealthChecks(t *testing.T) {
	agentInstance := createTestAgent(t, "health-agent")

	// Start agent
	go agentInstance.Start(":18083")
	time.Sleep(500 * time.Millisecond)
	defer agentInstance.Stop(context.Background())

	tests := []struct {
		name     string
		endpoint string
		wantCode int
	}{
		{"Liveness", "/health/live", http.StatusOK},
		{"Readiness", "/health/ready", http.StatusOK},
		{"Startup", "/health/startup", http.StatusOK},
		{"Overall Health", "/health", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get("http://localhost:18083" + tt.endpoint)
			if err != nil {
				t.Fatalf("Health check failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, resp.StatusCode)
			}
		})
	}
}

// TestAgentWithClient tests agent with SDK client
func TestAgentWithClient(t *testing.T) {
	ctx := context.Background()

	// Create and start agent
	agentInstance := createTestAgent(t, "client-agent")
	go agentInstance.Start(":18084")
	time.Sleep(500 * time.Millisecond)
	defer agentInstance.Stop(ctx)

	// Create client
	c, err := client.NewClient(
		"http://localhost:18084",
		client.WithProtocol(protocol.ProtocolA2A),
		client.WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Send message via client
	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello from client!"),
	})

	response, err := c.SendMessage(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.Role != types.MessageRoleAgent {
		t.Errorf("Expected role 'agent', got '%s'", response.Role)
	}
}

// createTestAgent creates a test agent with basic configuration
func createTestAgent(t *testing.T, name string) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		// Echo the message back
		return msg.Reply(fmt.Sprintf("Echo: %s", msg.Text()))
	}

	b := builder.NewAgent(name).
		WithDescription("Test agent for E2E testing").
		WithVersion("1.0.0").
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		t.Fatalf("Failed to build agent: %v", err)
		return nil
	}

	impl, ok := agentInstance.(*agent.AgentImpl)
	if !ok {
		t.Fatal("Agent is not *agent.AgentImpl")
		return nil
	}

	return impl
}
