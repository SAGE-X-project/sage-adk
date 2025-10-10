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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// NetworkClient handles HTTP communication for SAGE protocol.
type NetworkClient struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NetworkConfig configures the network client.
type NetworkConfig struct {
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

// DefaultNetworkConfig returns default network configuration.
func DefaultNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      1 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
}

// NewNetworkClient creates a new network client.
func NewNetworkClient(config *NetworkConfig) *NetworkClient {
	if config == nil {
		config = DefaultNetworkConfig()
	}

	return &NetworkClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        config.MaxIdleConns,
				IdleConnTimeout:     config.IdleConnTimeout,
				DisableCompression:  false,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 10,
			},
		},
		timeout: config.Timeout,
	}
}

// SendMessage sends a message to the remote endpoint via HTTP POST.
func (nc *NetworkClient) SendMessage(ctx context.Context, endpoint string, msg *types.Message) error {
	if endpoint == "" {
		return errors.ErrInvalidInput.WithMessage("endpoint cannot be empty")
	}

	if msg == nil {
		return errors.ErrInvalidInput.WithMessage("message cannot be nil")
	}

	// Serialize message to JSON
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to marshal message").
			WithDetail("error", err.Error())
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(messageBytes))
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to create HTTP request").
			WithDetail("error", err.Error())
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "sage-adk/1.0")

	// Add SAGE protocol headers
	if msg.Security != nil {
		req.Header.Set("X-SAGE-Protocol-Mode", string(msg.Security.Mode))
		req.Header.Set("X-SAGE-Agent-DID", msg.Security.AgentDID)
		req.Header.Set("X-SAGE-Nonce", msg.Security.Nonce)
		req.Header.Set("X-SAGE-Timestamp", msg.Security.Timestamp.Format(time.RFC3339))
	}

	// Send request
	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return errors.ErrOperationFailed.
			WithMessage("failed to send HTTP request").
			WithDetail("endpoint", endpoint).
			WithDetail("error", err.Error())
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return errors.ErrOperationFailed.
			WithMessage("HTTP request failed").
			WithDetail("status_code", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", string(bodyBytes))
	}

	return nil
}

// ReceiveMessage receives a message from HTTP request body.
func (nc *NetworkClient) ReceiveMessage(req *http.Request) (*types.Message, error) {
	if req == nil {
		return nil, errors.ErrInvalidInput.WithMessage("request cannot be nil")
	}

	// Read request body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to read request body").
			WithDetail("error", err.Error())
	}
	defer req.Body.Close()

	// Deserialize message
	var msg types.Message
	if err := json.Unmarshal(bodyBytes, &msg); err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to unmarshal message").
			WithDetail("error", err.Error())
	}

	return &msg, nil
}

// Close closes the network client and releases resources.
func (nc *NetworkClient) Close() error {
	nc.httpClient.CloseIdleConnections()
	return nil
}

// MessageHandler is a function type for handling incoming messages.
type MessageHandlerFunc func(ctx context.Context, msg *types.Message) (*types.Message, error)

// NetworkServer provides HTTP server for receiving SAGE messages.
type NetworkServer struct {
	httpServer *http.Server
	handler    MessageHandlerFunc
}

// NewNetworkServer creates a new network server.
func NewNetworkServer(addr string, handler MessageHandlerFunc) *NetworkServer {
	ns := &NetworkServer{
		handler: handler,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sage/message", ns.handleMessage)
	mux.HandleFunc("/health", ns.handleHealth)

	ns.httpServer = &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return ns
}

// handleMessage handles incoming SAGE messages.
func (ns *NetworkServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and deserialize message
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var msg types.Message
	if err := json.Unmarshal(bodyBytes, &msg); err != nil {
		http.Error(w, "Failed to parse message", http.StatusBadRequest)
		return
	}

	// Call handler
	if ns.handler != nil {
		response, err := ns.handler(r.Context(), &msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send response if handler returned one
		if response != nil {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusAccepted)
		}
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}

// handleHealth handles health check requests.
func (ns *NetworkServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "sage-adk",
	})
}

// Start starts the HTTP server.
func (ns *NetworkServer) Start() error {
	return ns.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server.
func (ns *NetworkServer) Stop(ctx context.Context) error {
	return ns.httpServer.Shutdown(ctx)
}

// Addr returns the server address.
func (ns *NetworkServer) Addr() string {
	return ns.httpServer.Addr
}
