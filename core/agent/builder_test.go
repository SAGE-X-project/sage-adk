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

package agent

import (
	"context"
	"testing"
)

func TestNewAgent(t *testing.T) {
	builder := NewAgent("test-agent")

	if builder == nil {
		t.Fatal("NewAgent() should not return nil")
	}
}

func TestBuilder_Build_Success(t *testing.T) {
	agent, err := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return msg.Reply("ok")
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}

	if agent == nil {
		t.Fatal("Build() should not return nil agent")
	}

	if agent.Name() != "test" {
		t.Errorf("Name() = %v, want test", agent.Name())
	}
}

func TestBuilder_Build_MissingHandler(t *testing.T) {
	_, err := NewAgent("test").Build()

	if err == nil {
		t.Error("Build() should return error when message handler is missing")
	}
}

func TestBuilder_Build_EmptyName(t *testing.T) {
	_, err := NewAgent("").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return nil
		}).
		Build()

	if err == nil {
		t.Error("Build() should return error when name is empty")
	}
}

func TestBuilder_WithDescription(t *testing.T) {
	agent, err := NewAgent("test").
		WithDescription("Test agent").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent.Description() != "Test agent" {
		t.Errorf("Description() = %v, want 'Test agent'", agent.Description())
	}
}

func TestBuilder_WithVersion(t *testing.T) {
	version := "1.2.3"

	agent, err := NewAgent("test").
		WithVersion(version).
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	card := agent.Card()
	if card.Version != version {
		t.Errorf("Card().Version = %v, want %v", card.Version, version)
	}
}

func TestBuilder_Chaining(t *testing.T) {
	// Test that builder methods return Builder for chaining
	builder := NewAgent("test").
		WithDescription("desc").
		WithVersion("1.0.0").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return nil
		})

	agent, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if agent == nil {
		t.Fatal("Build() should not return nil")
	}
}

func TestBuilder_DefaultConfig(t *testing.T) {
	agent, err := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cfg := agent.Config()
	if cfg == nil {
		t.Fatal("Config() should not return nil")
	}

	// Check default values
	if cfg.Agent.Name == "" {
		t.Error("Config should have default agent name")
	}

	if cfg.Server.Port == 0 {
		t.Error("Config should have default server port")
	}
}
