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

package a2a

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/core/agent"
)

func TestNewServer(t *testing.T) {
	config := &ServerConfig{
		AgentName:   "test-agent",
		AgentURL:    "http://localhost:8080/",
		Description: "A test agent",
		MessageHandler: func(ctx context.Context, msg agent.MessageContext) error {
			return nil
		},
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}

	if server.server == nil {
		t.Fatal("NewServer() created server with nil underlying server")
	}

	if server.handler == nil {
		t.Fatal("NewServer() created server with nil handler")
	}
}

func TestNewServer_NilHandler(t *testing.T) {
	config := &ServerConfig{
		AgentName:      "test-agent",
		AgentURL:       "http://localhost:8080/",
		Description:    "A test agent",
		MessageHandler: nil,
	}

	// Should not panic with nil handler
	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("NewServer() with nil handler failed: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}
}

func TestNewTaskManager(t *testing.T) {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		return nil
	}

	tm := newTaskManager(handler)
	if tm == nil {
		t.Fatal("newTaskManager() returned nil")
	}
}
