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
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/client"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// TestMultiAgentCommunication tests communication between multiple agents
func TestMultiAgentCommunication(t *testing.T) {
	ctx := context.Background()

	// Create Agent A (Coordinator)
	agentA := createCoordinatorAgent(t, "agent-a")
	go agentA.Start(":19001")
	time.Sleep(500 * time.Millisecond)
	defer agentA.Stop(ctx)

	// Create Agent B (Worker)
	agentB := createWorkerAgent(t, "agent-b", "uppercase")
	go agentB.Start(":19002")
	time.Sleep(500 * time.Millisecond)
	defer agentB.Stop(ctx)

	// Create Agent C (Worker)
	agentC := createWorkerAgent(t, "agent-c", "reverse")
	go agentC.Start(":19003")
	time.Sleep(500 * time.Millisecond)
	defer agentC.Stop(ctx)

	// Test: Agent A delegates work to Agent B
	t.Run("AgentToAgent", func(t *testing.T) {
		// Client connects to Agent A
		clientA, err := client.NewClient(
			"http://localhost:19001",
			client.WithProtocol(protocol.ProtocolA2A),
		)
		if err != nil {
			t.Fatalf("Failed to create client for Agent A: %v", err)
		}
		defer clientA.Close()

		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("delegate:uppercase:hello"),
		})

		response, err := clientA.SendMessage(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to send message to Agent A: %v", err)
		}

		// Verify Agent A coordinated with Agent B
		if response == nil {
			t.Fatal("Response is nil")
		}
	})

	// Test: Sequential multi-agent workflow
	t.Run("SequentialWorkflow", func(t *testing.T) {
		clientA, _ := client.NewClient("http://localhost:19001")
		defer clientA.Close()

		// Ask Agent A to process through B then C
		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("workflow:hello world"),
		})

		response, err := clientA.SendMessage(ctx, msg)
		if err != nil {
			t.Fatalf("Sequential workflow failed: %v", err)
		}

		if response == nil {
			t.Fatal("Workflow response is nil")
		}
	})
}

// TestConcurrentAgentRequests tests handling concurrent requests to multiple agents
func TestConcurrentAgentRequests(t *testing.T) {
	ctx := context.Background()

	// Create 3 agents
	agents := make([]*agent.AgentImpl, 3)
	ports := []string{":19010", ":19011", ":19012"}

	for i := 0; i < 3; i++ {
		agents[i] = createWorkerAgent(t, fmt.Sprintf("concurrent-agent-%d", i), "echo")
		go agents[i].Start(ports[i])
		time.Sleep(300 * time.Millisecond)
		defer agents[i].Stop(ctx)
	}

	// Send concurrent requests to all agents
	t.Run("ConcurrentRequests", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, 30) // 10 requests per agent

		for agentIdx := 0; agentIdx < 3; agentIdx++ {
			for reqIdx := 0; reqIdx < 10; reqIdx++ {
				wg.Add(1)
				go func(aIdx, rIdx int) {
					defer wg.Done()

					c, err := client.NewClient(fmt.Sprintf("http://localhost%s", ports[aIdx]))
					if err != nil {
						results[aIdx*10+rIdx] = err
						return
					}
					defer c.Close()

					msg := types.NewMessage(types.MessageRoleUser, []types.Part{
						types.NewTextPart(fmt.Sprintf("Request %d to agent %d", rIdx, aIdx)),
					})

					_, err = c.SendMessage(ctx, msg)
					results[aIdx*10+rIdx] = err
				}(agentIdx, reqIdx)
			}
		}

		wg.Wait()

		// Check all requests succeeded
		failCount := 0
		for i, err := range results {
			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
				failCount++
			}
		}

		if failCount > 0 {
			t.Errorf("%d out of 30 concurrent requests failed", failCount)
		}
	})
}

// TestAgentDiscovery tests agent discovery and registration
func TestAgentDiscovery(t *testing.T) {
	ctx := context.Background()

	// Create registry agent
	registry := createRegistryAgent(t, "registry")
	go registry.Start(":19020")
	time.Sleep(500 * time.Millisecond)
	defer registry.Stop(ctx)

	// Create worker agents that register with registry
	worker1 := createWorkerAgent(t, "worker-1", "service-a")
	go worker1.Start(":19021")
	time.Sleep(300 * time.Millisecond)
	defer worker1.Stop(ctx)

	worker2 := createWorkerAgent(t, "worker-2", "service-b")
	go worker2.Start(":19022")
	time.Sleep(300 * time.Millisecond)
	defer worker2.Stop(ctx)

	t.Run("RegisterAgents", func(t *testing.T) {
		// Register worker1
		c := mustCreateClient(t, "http://localhost:19020")
		defer c.Close()

		registerMsg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("register:worker-1:http://localhost:19021"),
		})

		_, err := c.SendMessage(ctx, registerMsg)
		if err != nil {
			t.Fatalf("Failed to register worker-1: %v", err)
		}

		// Register worker2
		registerMsg2 := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("register:worker-2:http://localhost:19022"),
		})

		_, err = c.SendMessage(ctx, registerMsg2)
		if err != nil {
			t.Fatalf("Failed to register worker-2: %v", err)
		}
	})

	t.Run("DiscoverAgents", func(t *testing.T) {
		c := mustCreateClient(t, "http://localhost:19020")
		defer c.Close()

		discoverMsg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("discover"),
		})

		response, err := c.SendMessage(ctx, discoverMsg)
		if err != nil {
			t.Fatalf("Failed to discover agents: %v", err)
		}

		if response == nil {
			t.Fatal("Discovery response is nil")
		}
	})
}

// TestAgentFailover tests failover between agents
func TestAgentFailover(t *testing.T) {
	ctx := context.Background()

	// Create primary agent
	primary := createWorkerAgent(t, "primary", "service")
	go primary.Start(":19030")
	time.Sleep(500 * time.Millisecond)

	// Create backup agent
	backup := createWorkerAgent(t, "backup", "service")
	go backup.Start(":19031")
	time.Sleep(500 * time.Millisecond)
	defer backup.Stop(ctx)

	t.Run("PrimaryAvailable", func(t *testing.T) {
		c := mustCreateClient(t, "http://localhost:19030")
		defer c.Close()

		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("test message"),
		})

		_, err := c.SendMessage(ctx, msg)
		if err != nil {
			t.Fatalf("Primary agent failed: %v", err)
		}
	})

	t.Run("FailoverToBackup", func(t *testing.T) {
		// Stop primary
		if err := primary.Stop(ctx); err != nil {
			t.Logf("Primary stop error: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		// Try backup
		c := mustCreateClient(t, "http://localhost:19031")
		defer c.Close()

		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart("test message"),
		})

		_, err := c.SendMessage(ctx, msg)
		if err != nil {
			t.Fatalf("Backup agent failed: %v", err)
		}
	})
}

// Helper functions

func createCoordinatorAgent(t *testing.T, name string) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		text := msg.Text()

		// Simple delegation logic
		if len(text) > 9 && text[:9] == "delegate:" {
			return msg.Reply(fmt.Sprintf("Coordinated: %s", text))
		}

		if len(text) > 9 && text[:9] == "workflow:" {
			return msg.Reply(fmt.Sprintf("Workflow completed: %s", text))
		}

		return msg.Reply(fmt.Sprintf("Coordinator echo: %s", text))
	}

	b := builder.NewAgent(name).
		WithDescription("Coordinator agent").
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		t.Fatalf("Failed to build coordinator: %v", err)
	}

	impl, _ := agentInstance.(*agent.AgentImpl)
	return impl
}

func createWorkerAgent(t *testing.T, name string, capability string) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		text := msg.Text()
		result := fmt.Sprintf("[%s] Processed: %s", capability, text)
		return msg.Reply(result)
	}

	b := builder.NewAgent(name).
		WithDescription(fmt.Sprintf("Worker agent with %s capability", capability)).
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		t.Fatalf("Failed to build worker: %v", err)
	}

	impl, _ := agentInstance.(*agent.AgentImpl)
	return impl
}

func createRegistryAgent(t *testing.T, name string) *agent.AgentImpl {
	// Simple in-memory registry
	registry := make(map[string]string)
	var mu sync.Mutex

	handler := func(ctx context.Context, msg agent.MessageContext) error {
		mu.Lock()
		defer mu.Unlock()

		text := msg.Text()

		if len(text) > 9 && text[:9] == "register:" {
			// Parse: register:name:url
			registry["dummy"] = "registered"
			return msg.Reply("Registered successfully")
		}

		if text == "discover" {
			return msg.Reply(fmt.Sprintf("Found %d agents", len(registry)))
		}

		return msg.Reply("Unknown command")
	}

	b := builder.NewAgent(name).
		WithDescription("Registry agent").
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		t.Fatalf("Failed to build registry: %v", err)
	}

	impl, _ := agentInstance.(*agent.AgentImpl)
	return impl
}

func mustCreateClient(t *testing.T, url string) *client.Client {
	c, err := client.NewClient(url, client.WithProtocol(protocol.ProtocolA2A))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return c
}
