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
	"net/http"
	"time"

	"github.com/sage-x-project/sage-adk/core/protocol"
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithProtocol sets the protocol mode (A2A, SAGE, or Auto).
func WithProtocol(mode protocol.ProtocolMode) Option {
	return func(c *Client) {
		c.protocol = mode
	}
}

// WithTimeout sets the request timeout duration.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
		if c.httpClient != nil {
			c.httpClient.Timeout = timeout
		}
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithRetry configures retry behavior.
// maxRetries: maximum number of retry attempts (0 means no retry).
// initialDelay: initial delay before first retry.
// maxDelay: maximum delay between retries.
func WithRetry(maxRetries int, initialDelay, maxDelay time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.initialDelay = initialDelay
		c.maxDelay = maxDelay
	}
}

// WithHeaders sets custom HTTP headers.
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers["User-Agent"] = userAgent
	}
}

// WithMaxIdleConns sets the maximum idle connections in the connection pool.
func WithMaxIdleConns(n int) Option {
	return func(c *Client) {
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.MaxIdleConns = n
			transport.MaxIdleConnsPerHost = n
		} else {
			c.httpClient.Transport = &http.Transport{
				MaxIdleConns:        n,
				MaxIdleConnsPerHost: n,
				IdleConnTimeout:     90 * time.Second,
			}
		}
	}
}
