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

package sage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNetworkClient_SendMessage(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Check SAGE headers
		if r.Header.Get("X-SAGE-Protocol-Mode") == "" {
			t.Error("Expected X-SAGE-Protocol-Mode header")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	client := NewNetworkClient(nil)

	// Create test message
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:test:0x123",
		Nonce:     "test-nonce",
		Timestamp: time.Now(),
	}

	// Send message
	err := client.SendMessage(context.Background(), server.URL, msg)
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}
}

func TestNetworkClient_SendMessage_InvalidEndpoint(t *testing.T) {
	client := NewNetworkClient(nil)

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Test with empty endpoint
	err := client.SendMessage(context.Background(), "", msg)
	if err == nil {
		t.Error("SendMessage() should return error for empty endpoint")
	}
}

func TestNetworkClient_SendMessage_NilMessage(t *testing.T) {
	client := NewNetworkClient(nil)

	err := client.SendMessage(context.Background(), "http://test.com", nil)
	if err == nil {
		t.Error("SendMessage() should return error for nil message")
	}
}

func TestNetworkClient_SendMessage_NetworkError(t *testing.T) {
	client := NewNetworkClient(nil)

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Test with invalid endpoint
	err := client.SendMessage(context.Background(), "http://invalid-endpoint-12345.local", msg)
	if err == nil {
		t.Error("SendMessage() should return error for network failure")
	}
}

func TestNetworkClient_SendMessage_HTTPError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewNetworkClient(nil)

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	err := client.SendMessage(context.Background(), server.URL, msg)
	if err == nil {
		t.Error("SendMessage() should return error for HTTP error response")
	}
}

func TestNetworkServer_HandleMessage(t *testing.T) {
	// Create test handler
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Handler receives message
		return nil, nil
	}

	// Create server
	server := NewNetworkServer(":0", handler)

	// Simulate HTTP request with invalid body
	req := httptest.NewRequest(http.MethodPost, "/sage/message", nil)
	w := httptest.NewRecorder()

	// Manually call handler
	server.handleMessage(w, req)

	// For invalid body, should return error
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestNetworkServer_Health(t *testing.T) {
	server := NewNetworkServer(":0", nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestNetworkClient_Close(t *testing.T) {
	client := NewNetworkClient(nil)

	err := client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDefaultNetworkConfig(t *testing.T) {
	config := DefaultNetworkConfig()

	if config == nil {
		t.Fatal("DefaultNetworkConfig() returned nil")
	}

	if config.Timeout == 0 {
		t.Error("Default timeout should not be zero")
	}

	if config.MaxRetries == 0 {
		t.Error("Default max retries should not be zero")
	}
}

func TestNewNetworkClient_WithNilConfig(t *testing.T) {
	client := NewNetworkClient(nil)

	if client == nil {
		t.Fatal("NewNetworkClient() returned nil")
	}

	if client.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestNewNetworkClient_WithCustomConfig(t *testing.T) {
	config := &NetworkConfig{
		Timeout:         10 * time.Second,
		MaxRetries:      5,
		RetryDelay:      2 * time.Second,
		MaxIdleConns:    50,
		IdleConnTimeout: 60 * time.Second,
	}

	client := NewNetworkClient(config)

	if client == nil {
		t.Fatal("NewNetworkClient() returned nil")
	}

	if client.timeout != config.Timeout {
		t.Errorf("Timeout = %v, want %v", client.timeout, config.Timeout)
	}
}
