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

	"github.com/sage-x-project/sage-adk/core/agent"
	a2aserver "trpc.group/trpc-go/trpc-a2a-go/server"
	"trpc.group/trpc-go/trpc-a2a-go/taskmanager"
)

// Server wraps the sage-a2a-go server for use in SAGE ADK.
//
// It integrates with the agent's message handler to process incoming requests.
type Server struct {
	server  *a2aserver.A2AServer
	handler agent.MessageHandler
}

// ServerConfig configures the A2A server.
type ServerConfig struct {
	// AgentName is the name of the agent
	AgentName string

	// AgentURL is the public URL of the agent
	AgentURL string

	// Description is a human-readable description
	Description string

	// MessageHandler processes incoming messages
	MessageHandler agent.MessageHandler
}

// NewServer creates a new A2A server.
//
// Example:
//
//	server, err := a2a.NewServer(&a2a.ServerConfig{
//	    AgentName: "chatbot",
//	    AgentURL:  "http://localhost:8080/",
//	    Description: "A simple chatbot agent",
//	    MessageHandler: func(ctx context.Context, msg agent.MessageContext) error {
//	        // Handle message
//	        return nil
//	    },
//	})
func NewServer(config *ServerConfig) (*Server, error) {
	// Create agent card
	agentCard := a2aserver.AgentCard{
		Name:        config.AgentName,
		URL:         config.AgentURL,
		Description: config.Description,
		Version:     "0.1.0",
	}

	// Create task manager with the message handler
	taskMgr := newTaskManager(config.MessageHandler)

	// Create A2A server
	a2aServer, err := a2aserver.NewA2AServer(agentCard, taskMgr)
	if err != nil {
		return nil, err
	}

	return &Server{
		server:  a2aServer,
		handler: config.MessageHandler,
	}, nil
}

// Start starts the HTTP server on the given address.
//
// This is a blocking call that returns when the server stops.
func (s *Server) Start(addr string) error {
	return s.server.Start(addr)
}

// Stop gracefully stops the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Stop(ctx)
}

// taskManager implements the taskmanager.TaskManager interface
// to integrate with the agent's message handler.
type taskManagerImpl struct {
	handler agent.MessageHandler
}

// newTaskManager creates a new task manager.
func newTaskManager(handler agent.MessageHandler) taskmanager.TaskManager {
	// Use the memory task manager from sage-a2a-go
	// and wrap the agent's message handler
	memoryTM := taskmanager.NewMemoryTaskManager(
		taskmanager.WithMessageHandler(func(ctx context.Context, msg taskmanager.MessageContext) error {
			// Convert to agent.MessageContext and call the agent's handler
			agentMsg := agent.MessageContext{
				Message: msg.GetMessage(),
			}
			return handler(ctx, agentMsg)
		}),
	)

	return memoryTM
}
