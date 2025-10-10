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

/*
Package client provides an HTTP client for communicating with SAGE ADK agents.

# Overview

The client package offers a simple and robust way to interact with SAGE ADK
agent servers. It supports both A2A (Agent-to-Agent) and SAGE (Secure Agent
Guarantee Engine) protocols, with features like automatic retry, connection
pooling, and streaming responses.

# Features

  - Protocol Support: A2A, SAGE, and automatic protocol detection
  - Retry Logic: Exponential backoff with configurable retries
  - Connection Pooling: Efficient connection reuse
  - Streaming: Server-Sent Events (SSE) for real-time responses
  - Context Support: Full context.Context integration for cancellation
  - Error Handling: Typed errors for better error handling
  - Customization: Extensive configuration via functional options

# Quick Start

Basic usage:

	client, err := client.NewClient("http://localhost:8080")
	if err != nil {
	    log.Fatal(err)
	}
	defer client.Close()

	// Create a message
	msg := &types.Message{
	    MessageID: "msg-123",
	    Role:      types.MessageRoleUser,
	    Parts: []types.Part{
	        &types.TextPart{
	            Kind: "text",
	            Text: "Hello, agent!",
	        },
	    },
	}

	// Send message
	response, err := client.SendMessage(context.Background(), msg)
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Println(response)

# Configuration

Configure the client with functional options:

	client, err := client.NewClient(
	    "http://localhost:8080",
	    client.WithProtocol(protocol.ProtocolSAGE),
	    client.WithTimeout(60*time.Second),
	    client.WithRetry(5, 200*time.Millisecond, 10*time.Second),
	    client.WithMaxIdleConns(50),
	    client.WithUserAgent("my-app/1.0.0"),
	)

# Protocol Modes

The client supports three protocol modes:

  - ProtocolAuto (default): Automatically detects protocol from message
  - ProtocolA2A: Forces A2A protocol for all messages
  - ProtocolSAGE: Forces SAGE protocol for all messages

Example:

	// Auto-detect protocol (default)
	client.SetProtocol(protocol.ProtocolAuto)

	// Force SAGE protocol
	client.SetProtocol(protocol.ProtocolSAGE)

# Streaming

Stream responses in real-time:

	events, err := client.StreamMessage(ctx, msg)
	if err != nil {
	    log.Fatal(err)
	}

	for chunk := range events {
	    switch chunk.Event {
	    case "message":
	        fmt.Println("Received:", chunk.Message)
	    case "error":
	        log.Printf("Error: %v", chunk.Error)
	    case "done":
	        fmt.Println("Stream completed")
	    }
	}

# Retry and Error Handling

The client automatically retries failed requests with exponential backoff:

	client, err := client.NewClient(
	    baseURL,
	    client.WithRetry(
	        3,                      // max retries
	        100*time.Millisecond,   // initial delay
	        5*time.Second,          // max delay
	    ),
	)

Error handling with typed errors:

	response, err := client.SendMessage(ctx, msg)
	if err != nil {
	    if errors.IsInvalidInput(err) {
	        log.Println("Invalid message format")
	    } else if errors.IsUnauthorized(err) {
	        log.Println("Authentication failed")
	    } else if errors.IsTimeout(err) {
	        log.Println("Request timed out")
	    } else {
	        log.Printf("Error: %v", err)
	    }
	    return
	}

# Connection Pooling

The client uses connection pooling for efficient HTTP communication:

	client, err := client.NewClient(
	    baseURL,
	    client.WithMaxIdleConns(100), // max idle connections
	)

# Custom HTTP Client

Use a custom HTTP client for advanced configuration:

	httpClient := &http.Client{
	    Transport: &http.Transport{
	        TLSClientConfig: &tls.Config{
	            InsecureSkipVerify: true,
	        },
	    },
	}

	client, err := client.NewClient(
	    baseURL,
	    client.WithHTTPClient(httpClient),
	)

# Thread Safety

The Client is safe for concurrent use by multiple goroutines.

# Best Practices

  - Always use context.Context for cancellation and timeouts
  - Close the client when done to release connections
  - Configure retry settings based on your use case
  - Use streaming for long-running operations
  - Handle errors appropriately with typed error checks

# See Also

  - pkg/types: Message types and structures
  - core/protocol: Protocol modes and adapters
  - pkg/errors: Error types and utilities
*/
package client
