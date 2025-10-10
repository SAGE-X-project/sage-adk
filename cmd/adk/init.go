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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new SAGE ADK project",
	Long: `Create a new SAGE ADK agent project with the specified name.

This command creates:
  - Project directory structure
  - main.go with agent setup
  - config.yaml configuration file
  - go.mod with required dependencies
  - README.md with getting started guide

Example:
  adk init my-agent
  adk init my-agent --protocol sage
  adk init my-agent --llm anthropic --storage redis`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

var (
	initProtocol string
	initLLM      string
	initStorage  string
)

func init() {
	initCmd.Flags().StringVar(&initProtocol, "protocol", "auto", "Protocol mode (auto, a2a, sage)")
	initCmd.Flags().StringVar(&initLLM, "llm", "openai", "LLM provider (openai, anthropic, gemini)")
	initCmd.Flags().StringVar(&initStorage, "storage", "memory", "Storage backend (memory, redis, postgres)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// Validate project name
	if projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	fmt.Printf("Creating SAGE ADK project '%s'...\n", projectName)

	// Create main.go
	if err := createMainGo(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created main.go")

	// Create config.yaml
	if err := createConfigYAML(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created config.yaml")

	// Create go.mod
	if err := createGoMod(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created go.mod")

	// Create README.md
	if err := createREADME(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created README.md")

	// Create .env.example
	if err := createEnvExample(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created .env.example")

	// Create .gitignore
	if err := createGitignore(projectName); err != nil {
		return err
	}
	fmt.Println("✓ Created .gitignore")

	fmt.Printf("\n✨ Project '%s' created successfully!\n\n", projectName)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  go mod tidy")
	fmt.Println("  cp .env.example .env")
	fmt.Println("  # Edit .env with your API keys")
	fmt.Println("  go run main.go")
	fmt.Println()

	return nil
}

func createMainGo(projectName string) error {
	content := getMainGoTemplate()
	return os.WriteFile(filepath.Join(projectName, "main.go"), []byte(content), 0644)
}

func createConfigYAML(projectName string) error {
	content := getConfigYAMLTemplate()
	return os.WriteFile(filepath.Join(projectName, "config.yaml"), []byte(content), 0644)
}

func createGoMod(projectName string) error {
	content := fmt.Sprintf(`module %s

go 1.21

require github.com/sage-x-project/sage-adk v1.0.0
`, projectName)
	return os.WriteFile(filepath.Join(projectName, "go.mod"), []byte(content), 0644)
}

func createREADME(projectName string) error {
	content := getREADMETemplate(projectName)
	return os.WriteFile(filepath.Join(projectName, "README.md"), []byte(content), 0644)
}

func createEnvExample(projectName string) error {
	content := getEnvExampleTemplate()
	return os.WriteFile(filepath.Join(projectName, ".env.example"), []byte(content), 0644)
}

func createGitignore(projectName string) error {
	content := `.env
*.log
/bin/
/dist/
.DS_Store
`
	return os.WriteFile(filepath.Join(projectName, ".gitignore"), []byte(content), 0644)
}

func getMainGoTemplate() string {
	llmProvider := strings.Title(initLLM)
	storageType := strings.Title(initStorage)

	return fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/storage"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func main() {
	// Load API key from environment
	apiKey := os.Getenv("%s_API_KEY")
	if apiKey == "" {
		log.Fatal("%s_API_KEY environment variable is required")
	}

	// Create storage
	store := storage.New%sStorage()

	// Create LLM provider
	llmProvider, err := llm.New%s(&llm.%sConfig{
		APIKey: apiKey,
		Model:  "default",
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %%v", err)
	}

	// Build agent
	agent, err := builder.NewAgent("my-agent").
		WithDescription("A helpful AI agent").
		WithLLM(llmProvider).
		WithStorage(store).
		OnMessage(handleMessage).
		Build()
	if err != nil {
		log.Fatalf("Failed to build agent: %%v", err)
	}

	fmt.Println("Agent started successfully!")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
}

func handleMessage(ctx context.Context, msg *types.Message) (*types.Message, error) {
	fmt.Printf("Received message: %%s\n", msg.MessageID)

	// Process message here
	response := types.Message{
		MessageID: "resp-" + msg.MessageID,
		Role:      types.MessageRoleAgent,
		Kind:      "message",
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "Hello! I received your message.",
			},
		},
	}

	return &response, nil
}
`, strings.ToUpper(initLLM), strings.ToUpper(initLLM), storageType, llmProvider, llmProvider)
}

func getConfigYAMLTemplate() string {
	return fmt.Sprintf(`# SAGE ADK Configuration

agent:
  name: my-agent
  version: 1.0.0
  protocol: %s

llm:
  provider: %s
  model: default
  temperature: 0.7
  max_tokens: 2000

storage:
  type: %s

server:
  host: 0.0.0.0
  port: 8080

observability:
  metrics_enabled: true
  logging_level: info
`, initProtocol, initLLM, initStorage)
}

func getREADMETemplate(projectName string) string {
	return fmt.Sprintf(`# %s

A SAGE ADK agent project.

## Setup

1. Install dependencies:
   `+"```"+`bash
   go mod tidy
   `+"```"+`

2. Configure environment:
   `+"```"+`bash
   cp .env.example .env
   # Edit .env with your API keys
   `+"```"+`

3. Run the agent:
   `+"```"+`bash
   go run main.go
   `+"```"+`

## Configuration

Edit `+"`config.yaml`"+` to customize your agent:

- **Protocol**: Choose between `+"`a2a`"+`, `+"`sage`"+`, or `+"`auto`"+`
- **LLM Provider**: OpenAI, Anthropic, or Gemini
- **Storage**: Memory, Redis, or PostgreSQL

## Documentation

- [SAGE ADK Documentation](https://github.com/sage-x-project/sage-adk)
- [API Reference](https://pkg.go.dev/github.com/sage-x-project/sage-adk)

## License

LGPL-3.0-or-later
`, projectName)
}

func getEnvExampleTemplate() string {
	return fmt.Sprintf(`# LLM API Keys
OPENAI_API_KEY=your-openai-api-key-here
ANTHROPIC_API_KEY=your-anthropic-api-key-here
GEMINI_API_KEY=your-gemini-api-key-here

# Storage Configuration
REDIS_URL=redis://localhost:6379
POSTGRES_URL=postgres://user:password@localhost/dbname

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Observability
LOG_LEVEL=info
METRICS_ENABLED=true
`)
}
