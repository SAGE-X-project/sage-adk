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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Client is an HTTP client for communicating with SAGE ADK agents.
// It supports both A2A and SAGE protocols, with features like retry logic,
// connection pooling, and streaming.
type Client struct {
	// baseURL is the base URL of the agent server (e.g., "http://localhost:8080")
	baseURL string

	// httpClient is the underlying HTTP client
	httpClient *http.Client

	// protocol is the protocol mode (A2A, SAGE, or Auto)
	protocol protocol.ProtocolMode

	// timeout is the request timeout
	timeout time.Duration

	// Retry configuration
	maxRetries   int
	initialDelay time.Duration
	maxDelay     time.Duration

	// headers are custom HTTP headers to include in requests
	headers map[string]string
}

// NewClient creates a new SAGE ADK client.
// baseURL should be the full URL to the agent server (e.g., "http://localhost:8080").
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, errors.ErrInvalidInput.WithMessage("baseURL cannot be empty")
	}

	// Remove trailing slash from baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Create default HTTP client with connection pooling
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	client := &Client{
		baseURL:      baseURL,
		httpClient:   httpClient,
		protocol:     protocol.ProtocolAuto,
		timeout:      30 * time.Second,
		maxRetries:   3,
		initialDelay: 100 * time.Millisecond,
		maxDelay:     5 * time.Second,
		headers:      make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// SendMessage sends a message to the agent and returns the response.
// It uses the configured protocol mode and automatically retries on failure.
func (c *Client) SendMessage(ctx context.Context, msg *types.Message) (*types.Message, error) {
	if msg == nil {
		return nil, errors.ErrInvalidInput.WithMessage("message cannot be nil")
	}

	// Validate message
	if err := msg.Validate(); err != nil {
		return nil, errors.ErrInvalidInput.WithMessage(fmt.Sprintf("invalid message: %v", err))
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(c.initialDelay) * math.Pow(2, float64(attempt-1)))
			if delay > c.maxDelay {
				delay = c.maxDelay
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		response, err := c.sendMessageOnce(ctx, msg)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		if errors.IsInvalidInput(err) || errors.IsUnauthorized(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

// sendMessageOnce sends a message once without retry.
func (c *Client) sendMessageOnce(ctx context.Context, msg *types.Message) (*types.Message, error) {
	// Marshal message to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, errors.ErrInvalidInput.WithMessage(fmt.Sprintf("failed to marshal message: %v", err))
	}

	// Create request
	url := c.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, respBody)
	}

	// Unmarshal response
	var response types.Message
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// handleErrorResponse converts HTTP error responses to appropriate errors.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &errResp)

	message := errResp.Message
	if message == "" {
		message = string(body)
	}

	switch statusCode {
	case http.StatusBadRequest:
		return errors.ErrInvalidInput.WithMessage(message)
	case http.StatusUnauthorized:
		return errors.ErrUnauthorized.WithMessage(message)
	case http.StatusNotFound:
		return errors.ErrNotFound.WithMessage(message)
	case http.StatusTooManyRequests:
		return errors.ErrRateLimitExceeded.WithMessage(message)
	case http.StatusInternalServerError:
		return errors.ErrInternal.WithMessage(message)
	case http.StatusServiceUnavailable:
		return errors.ErrInternal.WithMessage("service unavailable")
	case http.StatusGatewayTimeout:
		return errors.ErrTimeout.WithMessage("gateway timeout")
	default:
		return fmt.Errorf("HTTP %d: %s", statusCode, message)
	}
}

// SetProtocol changes the protocol mode.
func (c *Client) SetProtocol(mode protocol.ProtocolMode) {
	c.protocol = mode
}

// GetProtocol returns the current protocol mode.
func (c *Client) GetProtocol() protocol.ProtocolMode {
	return c.protocol
}

// Close closes the client and releases resources.
// It closes idle connections in the connection pool.
func (c *Client) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

// BaseURL returns the base URL of the agent server.
func (c *Client) BaseURL() string {
	return c.baseURL
}
