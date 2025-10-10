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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid URL",
			baseURL: "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "valid URL with trailing slash",
			baseURL: "http://localhost:8080/",
			wantErr: false,
		},
		{
			name:    "empty URL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "with options",
			baseURL: "http://localhost:8080",
			opts: []Option{
				WithProtocol(protocol.ProtocolSAGE),
				WithTimeout(60 * time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client == nil {
					t.Error("expected non-nil client")
				}
				defer client.Close()

				// Verify trailing slash removed
				if strings.HasSuffix(client.BaseURL(), "/") {
					t.Error("baseURL should not have trailing slash")
				}
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}

		// Read request body
		var msg types.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create response
		response := types.Message{
			MessageID: "resp-123",
			Role:      types.MessageRoleAgent,
			Kind:      "message",
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: "Hello from agent",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Create message
	msg := &types.Message{
		MessageID: "msg-123",
		Role:      types.MessageRoleUser,
		Kind:      "message",
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "Hello",
			},
		},
	}

	// Send message
	response, err := client.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	// Verify response
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if response.MessageID != "resp-123" {
		t.Errorf("expected message ID 'resp-123', got '%s'", response.MessageID)
	}
}

func TestClient_SendMessage_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Succeed on 3rd attempt
		response := types.Message{
			MessageID: "resp-123",
			Role:      types.MessageRoleAgent,
			Kind:      "message",
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: "Success",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with retry
	client, err := NewClient(
		server.URL,
		WithRetry(3, 10*time.Millisecond, 100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Create message
	msg := &types.Message{
		MessageID: "msg-123",
		Role:      types.MessageRoleUser,
		Kind:      "message",
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "Hello",
			},
		},
	}

	// Send message (should succeed after retries)
	response, err := client.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_SendMessage_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		errCheck   func(error) bool
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errCheck:   errors.IsInvalidInput,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errCheck:   errors.IsUnauthorized,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errCheck:   errors.IsNotFound,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
			errCheck:   errors.IsRateLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "test error",
					"message": "test error message",
				})
			}))
			defer server.Close()

			client, _ := NewClient(server.URL, WithRetry(0, 0, 0))
			defer client.Close()

			msg := &types.Message{
				MessageID: "msg-123",
				Role:      types.MessageRoleUser,
				Kind:      "message",
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: "Hello",
					},
				},
			}

			_, err := client.SendMessage(context.Background(), msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("error type check failed for error: %v", err)
			}
		})
	}
}

func TestClient_SendMessage_InvalidMessage(t *testing.T) {
	client, _ := NewClient("http://localhost:8080")
	defer client.Close()

	tests := []struct {
		name string
		msg  *types.Message
	}{
		{
			name: "nil message",
			msg:  nil,
		},
		{
			name: "empty message ID",
			msg: &types.Message{
				MessageID: "",
				Role:      types.MessageRoleUser,
				Kind:      "message",
				Parts:     []types.Part{},
			},
		},
		{
			name: "no parts",
			msg: &types.Message{
				MessageID: "msg-123",
				Role:      types.MessageRoleUser,
				Kind:      "message",
				Parts:     []types.Part{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendMessage(context.Background(), tt.msg)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestClient_ProtocolMode(t *testing.T) {
	client, _ := NewClient("http://localhost:8080")
	defer client.Close()

	// Default should be Auto
	if client.GetProtocol() != protocol.ProtocolAuto {
		t.Errorf("expected ProtocolAuto, got %v", client.GetProtocol())
	}

	// Set to SAGE
	client.SetProtocol(protocol.ProtocolSAGE)
	if client.GetProtocol() != protocol.ProtocolSAGE {
		t.Errorf("expected ProtocolSAGE, got %v", client.GetProtocol())
	}

	// Set to A2A
	client.SetProtocol(protocol.ProtocolA2A)
	if client.GetProtocol() != protocol.ProtocolA2A {
		t.Errorf("expected ProtocolA2A, got %v", client.GetProtocol())
	}
}

func TestClient_StreamMessage(t *testing.T) {
	// Create SSE test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/stream" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send message event
		msg := types.Message{
			MessageID: "stream-123",
			Role:      types.MessageRoleAgent,
			Kind:      "message",
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: "Streaming response",
				},
			},
		}
		msgJSON, _ := json.Marshal(msg)

		fmt.Fprintf(w, "event: message\n")
		fmt.Fprintf(w, "data: %s\n\n", msgJSON)
		flusher.Flush()

		// Send done event
		fmt.Fprintf(w, "event: done\n")
		fmt.Fprintf(w, "data: {}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client, _ := NewClient(server.URL)
	defer client.Close()

	msg := &types.Message{
		MessageID: "msg-123",
		Role:      types.MessageRoleUser,
		Kind:      "message",
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "Hello",
			},
		},
	}

	events, err := client.StreamMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("StreamMessage() error = %v", err)
	}

	receivedMessage := false
	receivedDone := false

	for chunk := range events {
		if chunk.Error != nil {
			t.Errorf("unexpected error in chunk: %v", chunk.Error)
		}

		switch chunk.Event {
		case "message":
			receivedMessage = true
			if chunk.Message == nil {
				t.Error("expected non-nil message")
			}
		case "done":
			receivedDone = true
		}
	}

	if !receivedMessage {
		t.Error("did not receive message event")
	}
	if !receivedDone {
		t.Error("did not receive done event")
	}
}

func TestClient_WithOptions(t *testing.T) {
	client, err := NewClient(
		"http://localhost:8080",
		WithProtocol(protocol.ProtocolSAGE),
		WithTimeout(60*time.Second),
		WithRetry(5, 200*time.Millisecond, 10*time.Second),
		WithHeaders(map[string]string{
			"X-Custom": "value",
		}),
		WithUserAgent("test-agent/1.0.0"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if client.GetProtocol() != protocol.ProtocolSAGE {
		t.Errorf("expected ProtocolSAGE, got %v", client.GetProtocol())
	}
	if client.timeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", client.timeout)
	}
	if client.maxRetries != 5 {
		t.Errorf("expected 5 retries, got %d", client.maxRetries)
	}
}

func TestClient_Context_Cancellation(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL)
	defer client.Close()

	msg := &types.Message{
		MessageID: "msg-123",
		Role:      types.MessageRoleUser,
		Kind:      "message",
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "Hello",
			},
		},
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.SendMessage(ctx, msg)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}
