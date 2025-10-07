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
	"encoding/base64"

	"github.com/sage-x-project/sage-adk/pkg/types"
	a2aclient "trpc.group/trpc-go/trpc-a2a-go/client"
	a2aprotocol "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Client wraps the sage-a2a-go client for simplified use in SAGE ADK.
type Client struct {
	client *a2aclient.A2AClient
}

// NewClient creates a new A2A client.
//
// Example:
//
//	client, err := a2a.NewClient("http://localhost:8080/")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewClient(serverURL string, opts ...a2aclient.Option) (*Client, error) {
	client, err := a2aclient.NewA2AClient(serverURL, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// SendMessage sends a message to the agent.
//
// This is a synchronous call that waits for the complete response.
func (c *Client) SendMessage(ctx context.Context, msg *types.Message) (*types.Message, error) {
	// Convert sage-adk message to A2A message
	a2aMsg := convertMessageToA2A(msg)

	// Send message
	params := a2aprotocol.SendMessageParams{
		Message: a2aMsg,
	}

	result, err := c.client.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert A2A message result back to sage-adk message
	// MessageResult.Result can be either *Message or *Task
	if resultMsg, ok := result.Result.(*a2aprotocol.Message); ok {
		return convertMessageFromA2A(resultMsg), nil
	}

	return nil, nil
}

// StreamMessage sends a message and receives streaming responses.
//
// The provided callback function will be called for each chunk of the response.
func (c *Client) StreamMessage(
	ctx context.Context,
	msg *types.Message,
	callback func(event types.StreamingEvent) error,
) error {
	// Convert sage-adk message to A2A message
	a2aMsg := convertMessageToA2A(msg)

	// Send streaming message
	params := a2aprotocol.SendMessageParams{
		Message: a2aMsg,
	}

	eventsChan, err := c.client.StreamMessage(ctx, params)
	if err != nil {
		return err
	}

	// Process events
	for event := range eventsChan {
		// Convert A2A event to sage-adk event
		sdkEvent := convertStreamingEventFromA2A(event)

		// Call user callback
		if err := callback(sdkEvent); err != nil {
			return err
		}
	}

	return nil
}

// convertMessageToA2A converts a sage-adk message to an A2A message.
func convertMessageToA2A(msg *types.Message) a2aprotocol.Message {
	parts := make([]a2aprotocol.Part, len(msg.Parts))
	for i, part := range msg.Parts {
		parts[i] = convertPartToA2A(part)
	}

	return a2aprotocol.Message{
		Role:  a2aprotocol.MessageRole(msg.Role),
		Parts: parts,
	}
}

// convertPartToA2A converts a sage-adk part to an A2A part.
func convertPartToA2A(part types.Part) a2aprotocol.Part {
	switch p := part.(type) {
	case *types.TextPart:
		return a2aprotocol.TextPart{
			Kind: "text",
			Text: p.Text,
		}
	case *types.FilePart:
		// Handle different file types
		if fileBytes, ok := p.File.(*types.FileWithBytes); ok {
			// Encode bytes to base64 string for A2A protocol
			base64Data := base64.StdEncoding.EncodeToString(fileBytes.Bytes)

			name := fileBytes.Name
			mimeType := fileBytes.MimeType

			return a2aprotocol.FilePart{
				Kind: "file",
				File: &a2aprotocol.FileWithBytes{
					Name:     &name,
					MimeType: &mimeType,
					Bytes:    base64Data,
				},
			}
		}
		if fileURI, ok := p.File.(*types.FileWithURI); ok {
			uri := fileURI.URI
			mimeType := fileURI.MimeType

			return a2aprotocol.FilePart{
				Kind: "file",
				File: &a2aprotocol.FileWithURI{
					URI:      uri,
					MimeType: &mimeType,
				},
			}
		}
	case *types.DataPart:
		return a2aprotocol.DataPart{
			Kind: "data",
			Data: p.Data,
		}
	}

	// Default: return text part with empty content
	return a2aprotocol.TextPart{
		Kind: "text",
		Text: "",
	}
}

// convertMessageFromA2A converts an A2A message to a sage-adk message.
func convertMessageFromA2A(msg *a2aprotocol.Message) *types.Message {
	parts := make([]types.Part, len(msg.Parts))
	for i, part := range msg.Parts {
		parts[i] = convertPartFromA2A(part)
	}

	return &types.Message{
		Role:  types.MessageRole(msg.Role),
		Parts: parts,
	}
}

// convertPartFromA2A converts an A2A part to a sage-adk part.
func convertPartFromA2A(part a2aprotocol.Part) types.Part {
	switch p := part.(type) {
	case a2aprotocol.TextPart:
		return types.NewTextPart(p.Text)

	case a2aprotocol.FilePart:
		// Handle different file representations
		if fileBytes, ok := p.File.(*a2aprotocol.FileWithBytes); ok {
			// Decode base64 string to bytes
			decodedBytes, err := base64.StdEncoding.DecodeString(fileBytes.Bytes)
			if err != nil {
				// If decoding fails, return empty text part
				return types.NewTextPart("")
			}

			name := ""
			if fileBytes.Name != nil {
				name = *fileBytes.Name
			}

			mimeType := ""
			if fileBytes.MimeType != nil {
				mimeType = *fileBytes.MimeType
			}

			return types.NewFilePartWithBytes(name, mimeType, decodedBytes)
		}
		if fileURI, ok := p.File.(*a2aprotocol.FileWithURI); ok {
			mimeType := ""
			if fileURI.MimeType != nil {
				mimeType = *fileURI.MimeType
			}

			return types.NewFilePartWithURI("", mimeType, fileURI.URI)
		}

	case a2aprotocol.DataPart:
		return types.NewDataPart(p.Data)
	}

	// Default: return empty text part
	return types.NewTextPart("")
}

// convertStreamingEventFromA2A converts an A2A streaming event to a sage-adk event.
func convertStreamingEventFromA2A(event a2aprotocol.StreamingMessageEvent) types.StreamingEvent {
	var msg *types.Message

	// StreamingMessageEvent.Result can be *Message, *Task, *TaskStatusUpdateEvent, or *TaskArtifactUpdateEvent
	if resultMsg, ok := event.Result.(*a2aprotocol.Message); ok {
		msg = convertMessageFromA2A(resultMsg)
	}

	return types.StreamingEvent{
		EventType: event.Result.GetKind(),
		Message:   msg,
	}
}
