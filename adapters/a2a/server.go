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
	"github.com/sage-x-project/sage-adk/pkg/types"
	a2aprotocol "trpc.group/trpc-go/trpc-a2a-go/protocol"
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

// messageProcessor implements the taskmanager.MessageProcessor interface
// to integrate with the agent's message handler.
type messageProcessor struct {
	handler agent.MessageHandler
}

// ProcessMessage processes an incoming message.
func (p *messageProcessor) ProcessMessage(
	ctx context.Context,
	msg a2aprotocol.Message,
	options taskmanager.ProcessOptions,
	handler taskmanager.TaskHandler,
) (*taskmanager.MessageProcessingResult, error) {
	// Convert A2A message to sage-adk message
	sdkMsg := convertA2AMessageToSDK(&msg)

	// For now, we create a simple messageContext wrapper
	// TODO: Implement proper MessageContext that supports Reply() and other methods
	msgCtx := &simpleMessageContext{
		message: sdkMsg,
	}

	// Call the agent's message handler
	if err := p.handler(ctx, msgCtx); err != nil {
		return nil, err
	}

	// For now, return a simple response message
	// In a real implementation, this would be the agent's actual response
	response := &a2aprotocol.Message{
		Role:  a2aprotocol.MessageRoleAgent,
		Parts: []a2aprotocol.Part{},
	}

	return &taskmanager.MessageProcessingResult{
		Result: response,
	}, nil
}

// simpleMessageContext is a simple implementation of MessageContext for the server.
type simpleMessageContext struct {
	message *types.Message
}

func (m *simpleMessageContext) Text() string {
	for _, part := range m.message.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			return textPart.Text
		}
	}
	return ""
}

func (m *simpleMessageContext) Parts() []types.Part {
	return m.message.Parts
}

func (m *simpleMessageContext) ContextID() string {
	if m.message.ContextID != nil {
		return *m.message.ContextID
	}
	return ""
}

func (m *simpleMessageContext) MessageID() string {
	return m.message.MessageID
}

func (m *simpleMessageContext) Reply(text string) error {
	// TODO: Implement reply functionality
	return nil
}

func (m *simpleMessageContext) ReplyWithParts(parts []types.Part) error {
	// TODO: Implement reply with parts functionality
	return nil
}

// convertA2AMessageToSDK converts an A2A protocol message to sage-adk message.
func convertA2AMessageToSDK(msg *a2aprotocol.Message) *types.Message {
	parts := make([]types.Part, len(msg.Parts))
	for i, part := range msg.Parts {
		parts[i] = convertPartFromA2A(part)
	}

	return &types.Message{
		Role:  types.MessageRole(msg.Role),
		Parts: parts,
	}
}

// newTaskManager creates a new task manager.
func newTaskManager(handler agent.MessageHandler) taskmanager.TaskManager {
	// Create message processor
	processor := &messageProcessor{
		handler: handler,
	}

	// Use the memory task manager from sage-a2a-go
	memoryTM, err := taskmanager.NewMemoryTaskManager(processor)
	if err != nil {
		// In case of error, return nil
		// This should be handled better in production code
		return nil
	}

	return memoryTM
}
