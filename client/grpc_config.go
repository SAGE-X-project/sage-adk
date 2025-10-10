// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"time"
)

// ClientConfig holds configuration for gRPC client
type ClientConfig struct {
	// Timeout for requests
	Timeout time.Duration

	// Retry configuration
	MaxRetries         int
	RetryInitialBackoff time.Duration
	RetryMaxBackoff    time.Duration
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:             30 * time.Second,
		MaxRetries:          3,
		RetryInitialBackoff: 100 * time.Millisecond,
		RetryMaxBackoff:     5 * time.Second,
	}
}

// ClientOption is a functional option for configuring gRPC client
type ClientOption func(*ClientConfig)

// WithClientTimeout sets the client timeout
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.Timeout = timeout
	}
}

// WithClientRetry configures client retry behavior
func WithClientRetry(maxRetries int, initialBackoff, maxBackoff time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.MaxRetries = maxRetries
		c.RetryInitialBackoff = initialBackoff
		c.RetryMaxBackoff = maxBackoff
	}
}
