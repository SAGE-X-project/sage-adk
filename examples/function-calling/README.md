# Function Calling Example

This example demonstrates how to use the SAGE ADK's function calling capabilities with LLM providers.

## Features Demonstrated

1. **Function/Tool Definition**: How to define functions that the LLM can call
2. **Tool Execution**: How to execute function calls and return results
3. **Token Counting**: How to estimate token usage
4. **Token Budget Management**: How to manage token budgets
5. **Message Truncation**: How to truncate conversation history to fit token limits

## Prerequisites

- OpenAI API key set in environment variable `OPENAI_API_KEY`
- Go 1.21 or later

## Running the Example

```bash
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key-here

# Run the example
cd sage-adk/examples/function-calling
go run main.go
```

## What It Does

1. **Creates two tools:**
   - `get_weather`: Gets weather for a location
   - `calculate`: Performs mathematical calculations

2. **Sends a query** that requires using both tools:
   - "What's the weather like in Tokyo? Also, what is 15 * 23?"

3. **Executes tool calls:**
   - Calls the weather service for Tokyo
   - Calls the calculator service for 15 * 23

4. **Demonstrates token management:**
   - Shows how to count tokens in text
   - Shows how to manage token budgets
   - Shows how to truncate messages to fit limits

## Code Structure

```go
// Define a function
function := llm.NewFunction(
    "get_weather",
    "Get the current weather for a location",
    llm.NewFunctionParameters().
        AddProperty("location", "string", "The city and state", true).
        AddEnumProperty("unit", "Temperature unit", []string{"celsius", "fahrenheit"}, false),
)

// Create a tool from the function
tool := llm.NewTool(function)

// Use the tool in a request
req := &llm.CompletionRequestWithTools{
    CompletionRequest: llm.CompletionRequest{
        Model: "gpt-4",
        Messages: messages,
    },
    Tools: []*llm.Tool{tool},
    ToolChoice: "auto",
}

// Call the LLM
resp, err := advProvider.CompleteWithTools(ctx, req)

// Process tool calls
for _, toolCall := range resp.ToolCalls {
    args, _ := toolCall.Function.ParsedArguments()
    // Execute your function with args
    result := executeFunction(toolCall.Function.Name, args)
    // Return result to LLM for final response
}
```

## Token Management Examples

### Token Counting
```go
counter := llm.NewSimpleTokenCounter()
tokens := counter.CountTokens("Your text here")
```

### Token Budget
```go
budget := llm.NewTokenBudget(counter, 1000)
if budget.CanAdd(text) {
    tokens := budget.Add(text)
    fmt.Printf("Remaining: %d\n", budget.Remaining())
}
```

### Message Truncation
```go
truncated := llm.TruncateMessages(messages, counter, maxTokens)
// Returns messages that fit within token limit
// Always keeps system message + most recent messages
```

## Function Calling Flow

```
1. User asks a question
   ↓
2. LLM decides to call functions
   ↓
3. Your code executes the functions
   ↓
4. Return results to LLM
   ↓
5. LLM generates final response
   (Step 4-5 not shown in this simplified example)
```

## Provider Support

| Provider   | Function Calling | Notes                      |
|------------|-----------------|----------------------------|
| OpenAI     | ✅ Yes          | Native tool support        |
| Anthropic  | ✅ Yes          | Uses tool_use content type |
| Gemini     | ✅ Yes          | Uses functionCall in parts |

## Advanced Features

### Custom Token Counters

```go
// Word-based counting (default)
simpleCounter := llm.NewSimpleTokenCounter()
simpleCounter.TokensPerWord = 1.3

// Character-based counting
charCounter := llm.NewCharacterBasedTokenCounter()
charCounter.CharsPerToken = 4.0
```

### Model Token Limits

```go
limit := llm.GetModelTokenLimit("gpt-4")
// Returns: 8192

limit := llm.GetModelTokenLimit("claude-3-opus")
// Returns: 200000

limit := llm.GetModelTokenLimit("gemini-1.5-pro")
// Returns: 1048576
```

## Error Handling

```go
resp, err := advProvider.CompleteWithTools(ctx, req)
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Check for tool calls
if len(resp.ToolCalls) > 0 {
    for _, tc := range resp.ToolCalls {
        args, err := tc.Function.ParsedArguments()
        if err != nil {
            log.Printf("Invalid arguments: %v", err)
            continue
        }
        // Execute function...
    }
}
```

## Notes

- This example shows a simplified flow where tool results are displayed but not sent back to the LLM
- In a real application, you would:
  1. Add the assistant's response with tool calls to conversation history
  2. Add tool results as tool messages to conversation history
  3. Call the LLM again to get a final natural language response
- Token counting is approximate; use provider-specific tokenizers for exact counts
- Always validate function arguments before execution
- Consider implementing timeouts and rate limiting for function calls

## Next Steps

- See `examples/streaming-agent/` for streaming with function calling
- See `examples/stateful-agent/` for managing conversation state
- See provider-specific examples for advanced features
