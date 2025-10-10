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

package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// StreamChunk represents a chunk of data received during streaming.
type StreamChunk struct {
	// Event is the event type (e.g., "message", "done", "error")
	Event string

	// Data is the raw event data
	Data string

	// Message is the parsed message (if Event is "message")
	Message *types.Message

	// Error is the error (if Event is "error")
	Error error
}

// StreamMessage sends a message to the agent and streams the response.
// It returns a channel that receives StreamChunk events as they arrive.
// The channel is closed when the stream ends or an error occurs.
//
// Example usage:
//
//	events, err := client.StreamMessage(ctx, message)
//	if err != nil {
//	    return err
//	}
//
//	for chunk := range events {
//	    if chunk.Error != nil {
//	        return chunk.Error
//	    }
//	    if chunk.Event == "message" {
//	        fmt.Println(chunk.Message)
//	    }
//	}
func (c *Client) StreamMessage(ctx context.Context, msg *types.Message) (<-chan *StreamChunk, error) {
	if msg == nil {
		return nil, errors.ErrInvalidInput.WithMessage("message cannot be nil")
	}

	// Validate message
	if err := msg.Validate(); err != nil {
		return nil, errors.ErrInvalidInput.WithMessage(fmt.Sprintf("invalid message: %v", err))
	}

	// Marshal message to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, errors.ErrInvalidInput.WithMessage(fmt.Sprintf("failed to marshal message: %v", err))
	}

	// Create request
	url := c.baseURL + "/v1/messages/stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Add protocol header
	req.Header.Set("X-Protocol-Mode", c.protocol.String())

	// Add custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// Create channel for streaming events
	events := make(chan *StreamChunk, 10)

	// Start goroutine to read SSE stream
	go c.readSSEStream(ctx, resp.Body, events)

	return events, nil
}

// readSSEStream reads Server-Sent Events from the response body.
func (c *Client) readSSEStream(ctx context.Context, body io.ReadCloser, events chan<- *StreamChunk) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var eventType string
	var eventData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check context cancellation
		select {
		case <-ctx.Done():
			events <- &StreamChunk{
				Event: "error",
				Error: ctx.Err(),
			}
			return
		default:
		}

		// Empty line indicates end of event
		if line == "" {
			if eventType != "" {
				chunk := c.parseSSEEvent(eventType, eventData.String())
				events <- chunk

				// Stop if error or done
				if chunk.Event == "error" || chunk.Event == "done" {
					return
				}
			}

			// Reset for next event
			eventType = ""
			eventData.Reset()
			continue
		}

		// Parse SSE fields
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if eventData.Len() > 0 {
				eventData.WriteString("\n")
			}
			eventData.WriteString(data)
		}
		// Ignore other fields (id:, retry:, etc.)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		events <- &StreamChunk{
			Event: "error",
			Error: fmt.Errorf("stream read error: %w", err),
		}
	}
}

// parseSSEEvent parses an SSE event into a StreamChunk.
func (c *Client) parseSSEEvent(eventType, data string) *StreamChunk {
	chunk := &StreamChunk{
		Event: eventType,
		Data:  data,
	}

	switch eventType {
	case "message":
		// Parse message from JSON
		var msg types.Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			chunk.Event = "error"
			chunk.Error = fmt.Errorf("failed to parse message: %w", err)
		} else {
			chunk.Message = &msg
		}

	case "error":
		// Parse error from JSON
		var errData struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &errData); err != nil {
			chunk.Error = fmt.Errorf("stream error: %s", data)
		} else {
			chunk.Error = fmt.Errorf("%s: %s", errData.Error, errData.Message)
		}

	case "done":
		// Stream completed successfully
		// No additional processing needed
	}

	return chunk
}
