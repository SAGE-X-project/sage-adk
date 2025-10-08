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

package builder

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage-adk/storage"
)

func TestNewAgent(t *testing.T) {
	builder := NewAgent("test-agent")

	if builder == nil {
		t.Fatal("NewAgent() returned nil")
	}

	if builder.name != "test-agent" {
		t.Errorf("name = %v, want test-agent", builder.name)
	}

	if builder.protocolMode != protocol.ProtocolA2A {
		t.Errorf("default protocol = %v, want ProtocolA2A", builder.protocolMode)
	}
}

func TestBuilder_Minimal_Success(t *testing.T) {
	// Minimal agent: just a name
	agent, err := NewAgent("minimal").Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}

	if agent.Name() != "minimal" {
		t.Errorf("Name() = %v, want minimal", agent.Name())
	}

	// Should have defaults
	if agent.Storage() == nil {
		t.Error("Storage() is nil, want default memory storage")
	}

	if agent.ProtocolMode() != protocol.ProtocolA2A {
		t.Errorf("ProtocolMode() = %v, want ProtocolA2A", agent.ProtocolMode())
	}
}

func TestBuilder_WithLLM(t *testing.T) {
	mockProvider := llm.NewMockProvider("test", []string{"response1"})

	agent, err := NewAgent("llm-agent").
		WithLLM(mockProvider).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent.LLMProvider() == nil {
		t.Error("LLMProvider() is nil")
	}

	if agent.LLMProvider().Name() != "test" {
		t.Errorf("LLMProvider().Name() = %v, want test", agent.LLMProvider().Name())
	}
}

func TestBuilder_WithProtocol(t *testing.T) {
	tests := []struct {
		name     string
		mode     protocol.ProtocolMode
		wantMode protocol.ProtocolMode
	}{
		{"A2A", protocol.ProtocolA2A, protocol.ProtocolA2A},
		{"Auto", protocol.ProtocolAuto, protocol.ProtocolAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip Auto protocol test until implemented
			if tt.mode == protocol.ProtocolAuto {
				t.Skip("Auto protocol mode not yet implemented")
			}

			agent, err := NewAgent("protocol-agent").
				WithProtocol(tt.mode).
				Build()

			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			if agent.ProtocolMode() != tt.wantMode {
				t.Errorf("ProtocolMode() = %v, want %v", agent.ProtocolMode(), tt.wantMode)
			}
		})
	}
}

func TestBuilder_WithSAGE_NoConfig_Error(t *testing.T) {
	// SAGE mode without config should fail
	_, err := NewAgent("sage-agent").
		WithProtocol(protocol.ProtocolSAGE).
		Build()

	if err == nil {
		t.Error("Build() should return error for SAGE mode without config")
	}
}

func TestBuilder_WithSAGE_WithConfig_Success(t *testing.T) {
	t.Skip("SAGE protocol server not yet implemented")

	sageConfig := &config.SAGEConfig{
		DID:         "did:sage:ethereum:0x123",
		Network:     "ethereum",
		RPCEndpoint: "http://localhost:8545",
	}

	agent, err := NewAgent("sage-agent").
		WithProtocol(protocol.ProtocolSAGE).
		WithSAGEConfig(sageConfig).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent.ProtocolMode() != protocol.ProtocolSAGE {
		t.Errorf("ProtocolMode() = %v, want ProtocolSAGE", agent.ProtocolMode())
	}
}

func TestBuilder_WithStorage(t *testing.T) {
	memStorage := storage.NewMemoryStorage()

	agent, err := NewAgent("storage-agent").
		WithStorage(memStorage).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent.Storage() == nil {
		t.Error("Storage() is nil")
	}
}

func TestBuilder_OnMessage(t *testing.T) {
	handlerCalled := false

	handler := func(ctx context.Context, msg agent.MessageContext) error {
		handlerCalled = true
		return nil
	}

	agent, err := NewAgent("handler-agent").
		OnMessage(handler).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Process a message to verify handler is set
	msg := &types.Message{
		MessageID: "test-123",
		Role:      "user",
	}

	_, _ = agent.Process(context.Background(), msg)

	if !handlerCalled {
		t.Error("Message handler was not called")
	}
}

func TestBuilder_BeforeStart(t *testing.T) {
	hookCalled := false

	hook := func(ctx context.Context) error {
		hookCalled = true
		return nil
	}

	agent, err := NewAgent("hook-agent").
		BeforeStart(hook).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Start will call the hook (but will fail because Start not implemented yet)
	// We just check the hook is set
	_ = agent.Start(":8080")

	if !hookCalled {
		t.Error("BeforeStart hook was not called")
	}
}

func TestBuilder_MustBuild_Success(t *testing.T) {
	// MustBuild should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked: %v", r)
		}
	}()

	agent := NewAgent("must-build-agent").MustBuild()

	if agent == nil {
		t.Error("MustBuild() returned nil")
	}
}

func TestBuilder_MustBuild_Panic(t *testing.T) {
	// MustBuild should panic with invalid config
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should panic with invalid config")
		}
	}()

	// Empty name should cause panic
	NewAgent("").MustBuild()
}

func TestBuilder_Validation_EmptyName(t *testing.T) {
	_, err := NewAgent("").Build()

	if err == nil {
		t.Error("Build() should return error for empty name")
	}
}

func TestBuilder_Validation_InvalidNameChars(t *testing.T) {
	_, err := NewAgent("invalid name!").Build()

	if err == nil {
		t.Error("Build() should return error for invalid name characters")
	}
}

func TestBuilder_Validation_NameTooLong(t *testing.T) {
	// Name longer than 64 characters
	longName := "this-is-a-very-long-agent-name-that-exceeds-the-maximum-length-limit-of-64-characters"

	_, err := NewAgent(longName).Build()

	if err == nil {
		t.Error("Build() should return error for name too long")
	}
}

func TestBuilder_Validation_ValidNames(t *testing.T) {
	validNames := []string{
		"agent",
		"my-agent",
		"my_agent",
		"agent123",
		"Agent-123_v2",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			_, err := NewAgent(name).Build()

			if err != nil {
				t.Errorf("Build() error = %v for valid name %s", err, name)
			}
		})
	}
}

func TestBuilder_FullyConfigured(t *testing.T) {
	// Test builder with all options
	mockProvider := llm.NewMockProvider("test", []string{"response"})
	memStorage := storage.NewMemoryStorage()

	var beforeStartCalled bool
	var afterStopCalled bool

	agent, err := NewAgent("full-agent").
		WithLLM(mockProvider).
		WithProtocol(protocol.ProtocolA2A).
		WithStorage(memStorage).
		OnMessage(func(ctx context.Context, msg agent.MessageContext) error {
			return nil
		}).
		BeforeStart(func(ctx context.Context) error {
			beforeStartCalled = true
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			afterStopCalled = true
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent.Name() != "full-agent" {
		t.Errorf("Name() = %v, want full-agent", agent.Name())
	}

	if agent.LLMProvider() == nil {
		t.Error("LLMProvider() is nil")
	}

	if agent.Storage() == nil {
		t.Error("Storage() is nil")
	}

	// Test hooks
	_ = agent.Start(":8080")
	if !beforeStartCalled {
		t.Error("BeforeStart hook not called")
	}

	_ = agent.Stop(context.Background())
	if !afterStopCalled {
		t.Error("AfterStop hook not called")
	}
}

func TestFromSAGEConfig(t *testing.T) {
	sageCfg := &config.SAGEConfig{
		Enabled:         true,
		DID:             "did:sage:sepolia:0x123",
		Network:         "sepolia",
		RPCEndpoint:     "https://sepolia.example.com",
		ContractAddress: "0xABC",
		PrivateKeyPath:  "./keys/test.pem",
	}

	builder := FromSAGEConfig(sageCfg)

	if builder == nil {
		t.Fatal("FromSAGEConfig() returned nil")
	}

	if builder.protocolMode != protocol.ProtocolSAGE {
		t.Errorf("protocol mode = %v, want ProtocolSAGE", builder.protocolMode)
	}

	if builder.sageConfig != sageCfg {
		t.Error("SAGE config not set")
	}

	if builder.config.SAGE.DID != sageCfg.DID {
		t.Error("SAGE config not set in main config")
	}

	// Test that builder can be further configured
	builder2 := FromSAGEConfig(sageCfg).
		WithLLM(llm.NewMockProvider("test", []string{"response"})).
		WithStorage(storage.NewMemoryStorage())

	if builder2.llmProvider == nil {
		t.Error("LLM provider not set")
	}

	if builder2.storageBackend == nil {
		t.Error("Storage backend not set")
	}
}

func TestBuilder_WithSAGEConfig(t *testing.T) {
	sageCfg := &config.SAGEConfig{
		Enabled:         true,
		DID:             "did:sage:ethereum:0x456",
		Network:         "ethereum",
		RPCEndpoint:     "https://eth.example.com",
		ContractAddress: "0xDEF",
		PrivateKeyPath:  "./keys/test2.pem",
	}

	builder := NewAgent("test-agent").
		WithProtocol(protocol.ProtocolSAGE).
		WithSAGEConfig(sageCfg)

	if builder.sageConfig != sageCfg {
		t.Error("SAGE config not set")
	}

	if builder.config.SAGE.DID != sageCfg.DID {
		t.Error("SAGE config not set in main config")
	}

	if builder.protocolMode != protocol.ProtocolSAGE {
		t.Errorf("protocol mode = %v, want ProtocolSAGE", builder.protocolMode)
	}
}

func TestBuilder_Idempotent(t *testing.T) {
	// Building multiple times should work
	builder := NewAgent("idempotent").
		WithLLM(llm.NewMockProvider("test", []string{"resp"}))

	agent1, err1 := builder.Build()
	agent2, err2 := builder.Build()

	if err1 != nil || err2 != nil {
		t.Fatalf("Build() errors = %v, %v", err1, err2)
	}

	if agent1 == nil || agent2 == nil {
		t.Fatal("Build() returned nil agents")
	}

	// Both should have same config
	if agent1.Name() != agent2.Name() {
		t.Error("Multiple builds produced different agents")
	}
}
