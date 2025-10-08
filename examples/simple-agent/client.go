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

//go:build examples
// +build examples

// Package main provides a simple client to test the chatbot agent
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sage-x-project/sage-adk/adapters/a2a"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run client.go <message>")
		fmt.Println("Example: go run client.go \"Hello, how are you?\"")
		os.Exit(1)
	}

	message := os.Args[1]

	// Create A2A client
	client, err := a2a.NewClient("http://localhost:8080/")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create message
	msg := &types.Message{
		MessageID: types.GenerateMessageID(),
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			types.NewTextPart(message),
		},
	}

	fmt.Printf("📤 Sending: %s\n", message)

	// Send message and get response
	response, err := client.SendMessage(context.Background(), msg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	// Print response
	fmt.Println("📥 Response:")
	for _, part := range response.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			fmt.Printf("   %s\n", textPart.Text)
		}
	}
}
