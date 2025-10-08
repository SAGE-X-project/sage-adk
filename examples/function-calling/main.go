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

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/llm"
)

// Weather API simulation
type WeatherService struct{}

func (ws *WeatherService) GetWeather(location string, unit string) (string, error) {
	// Simulate API call
	time.Sleep(100 * time.Millisecond)

	// Generate random weather
	conditions := []string{"sunny", "cloudy", "rainy", "snowy"}
	condition := conditions[rand.Intn(len(conditions))]

	temp := rand.Intn(30) + 10 // 10-40 range

	if unit == "fahrenheit" {
		temp = temp*9/5 + 32
	}

	return fmt.Sprintf("The weather in %s is %s with a temperature of %d°%s",
		location, condition, temp, unit[0:1]), nil
}

// Calculator service
type CalculatorService struct{}

func (cs *CalculatorService) Calculate(expression string) (float64, error) {
	// Simple calculator - just for demo
	// In real app, use proper expression parser
	return 42.0, nil
}

func main() {
	fmt.Println("=== SAGE ADK Function Calling Example ===\n")

	// Initialize services
	weatherService := &WeatherService{}
	calcService := &CalculatorService{}

	// Create LLM provider (OpenAI with function calling support)
	provider := llm.OpenAI(&llm.OpenAIConfig{
		Model: "gpt-4",
	})

	// Check if provider supports function calling
	advProvider, ok := provider.(llm.AdvancedProvider)
	if !ok || !advProvider.SupportsFunctionCalling() {
		log.Fatal("Provider does not support function calling")
	}

	// Define tools
	tools := []*llm.Tool{
		llm.NewTool(llm.NewFunction(
			"get_weather",
			"Get the current weather for a location",
			llm.NewFunctionParameters().
				AddProperty("location", "string", "The city and state, e.g. San Francisco, CA", true).
				AddEnumProperty("unit", "Temperature unit", []string{"celsius", "fahrenheit"}, false),
		)),
		llm.NewTool(llm.NewFunction(
			"calculate",
			"Perform a mathematical calculation",
			llm.NewFunctionParameters().
				AddProperty("expression", "string", "The mathematical expression to evaluate", true),
		)),
	}

	// Create request
	req := &llm.CompletionRequestWithTools{
		CompletionRequest: llm.CompletionRequest{
			Model: "gpt-4",
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: "You are a helpful assistant with access to tools."},
				{Role: llm.RoleUser, Content: "What's the weather like in Tokyo? Also, what is 15 * 23?"},
			},
			MaxTokens:   500,
			Temperature: 0.7,
		},
		Tools:      tools,
		ToolChoice: "auto",
	}

	fmt.Println("User: What's the weather like in Tokyo? Also, what is 15 * 23?\n")

	// Call LLM
	ctx := context.Background()
	resp, err := advProvider.CompleteWithTools(ctx, req)
	if err != nil {
		log.Fatalf("Error calling LLM: %v", err)
	}

	// Check for tool calls
	if len(resp.ToolCalls) > 0 {
		fmt.Printf("Assistant wants to call %d tool(s):\n\n", len(resp.ToolCalls))

		// Execute tool calls
		toolResults := make([]*llm.ToolCallResult, 0, len(resp.ToolCalls))

		for _, toolCall := range resp.ToolCalls {
			fmt.Printf("Tool: %s\n", toolCall.Function.Name)
			fmt.Printf("Arguments: %s\n", toolCall.Function.Arguments)

			// Parse arguments
			args, err := toolCall.Function.ParsedArguments()
			if err != nil {
				log.Printf("Error parsing arguments: %v", err)
				continue
			}

			// Execute function
			var result string
			switch toolCall.Function.Name {
			case "get_weather":
				location := args["location"].(string)
				unit := "celsius"
				if u, ok := args["unit"].(string); ok {
					unit = u
				}

				weather, err := weatherService.GetWeather(location, unit)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				} else {
					result = weather
				}

			case "calculate":
				expression := args["expression"].(string)
				calcResult, err := calcService.Calculate(expression)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				} else {
					result = fmt.Sprintf("%.2f", calcResult)
				}

			default:
				result = fmt.Sprintf("Unknown function: %s", toolCall.Function.Name)
			}

			fmt.Printf("Result: %s\n\n", result)

			// Add tool result
			toolResults = append(toolResults, &llm.ToolCallResult{
				ToolCallID: toolCall.ID,
				Role:       "tool",
				Content:    result,
				Name:       toolCall.Function.Name,
			})
		}

		// Send tool results back to LLM
		// Note: This requires extending the conversation with tool results
		// For this example, we'll just show the results
		fmt.Println("=== Tool Execution Complete ===")
		fmt.Println("\nIn a real implementation, you would:")
		fmt.Println("1. Add the assistant's tool calls to the conversation history")
		fmt.Println("2. Add the tool results to the conversation history")
		fmt.Println("3. Call the LLM again to get the final response")
		fmt.Println("\nTool Results:")
		for _, tr := range toolResults {
			fmt.Printf("- %s: %s\n", tr.Name, tr.Content)
		}
	} else {
		// No tool calls, just show response
		fmt.Printf("Assistant: %s\n", resp.Content)
	}

	// Show token usage
	if resp.Usage != nil {
		fmt.Printf("\n=== Token Usage ===\n")
		fmt.Printf("Prompt tokens: %d\n", resp.Usage.PromptTokens)
		fmt.Printf("Completion tokens: %d\n", resp.Usage.CompletionTokens)
		fmt.Printf("Total tokens: %d\n", resp.Usage.TotalTokens)
	}

	// Demonstrate token counting
	fmt.Printf("\n=== Token Counting Demo ===\n")
	text := "This is a sample text for token counting demonstration."
	simpleCount := advProvider.CountTokens(text)
	fmt.Printf("Text: %s\n", text)
	fmt.Printf("Estimated tokens: %d\n", simpleCount)

	// Demonstrate token budget
	fmt.Printf("\n=== Token Budget Demo ===\n")
	counter := llm.NewSimpleTokenCounter()
	budget := llm.NewTokenBudget(counter, 100)

	messages := []string{
		"Hello, how are you?",
		"I'm doing well, thank you!",
		"What can you help me with today?",
		"This message might exceed the budget if we keep adding more and more text...",
	}

	for i, msg := range messages {
		if budget.CanAdd(msg) {
			tokens := budget.Add(msg)
			fmt.Printf("Message %d added: %d tokens (remaining: %d)\n", i+1, tokens, budget.Remaining())
		} else {
			fmt.Printf("Message %d would exceed budget (remaining: %d)\n", i+1, budget.Remaining())
		}
	}

	// Demonstrate message truncation
	fmt.Printf("\n=== Message Truncation Demo ===\n")
	longMessages := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are a helpful assistant."},
		{Role: llm.RoleUser, Content: "What is AI?"},
		{Role: llm.RoleAssistant, Content: "AI stands for Artificial Intelligence..."},
		{Role: llm.RoleUser, Content: "Tell me more about machine learning."},
		{Role: llm.RoleAssistant, Content: "Machine learning is a subset of AI..."},
		{Role: llm.RoleUser, Content: "What about deep learning?"},
	}

	truncated := llm.TruncateMessages(longMessages, counter, 50)
	fmt.Printf("Original messages: %d\n", len(longMessages))
	fmt.Printf("Truncated messages: %d\n", len(truncated))
	fmt.Println("Kept messages:")
	for i, msg := range truncated {
		fmt.Printf("  %d. [%s] %s\n", i+1, msg.Role, msg.Content)
	}

	fmt.Println("\n=== Example Complete ===")
}
