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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/storage"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the SAGE ADK agent server",
	Long: `Start the HTTP server to serve your SAGE ADK agent.

The serve command starts an HTTP server that exposes your agent via REST API.
It supports both A2A and SAGE protocols, with built-in middleware for logging,
metrics, health checks, and more.

Configuration can be provided via:
  - config.yaml file (default: ./config.yaml)
  - Environment variables
  - Command-line flags (highest priority)

Example:
  adk serve
  adk serve --config my-config.yaml
  adk serve --port 9000 --host 0.0.0.0`,
	RunE: runServe,
}

var (
	serveConfig string
	servePort   int
	serveHost   string
)

func init() {
	serveCmd.Flags().StringVarP(&serveConfig, "config", "c", "config.yaml", "Path to configuration file")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Server port")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Server host")
}

func runServe(cmd *cobra.Command, args []string) error {
	log.Printf("🚀 Starting SAGE ADK server...")
	log.Printf("📄 Config: %s", serveConfig)
	log.Printf("🌐 Address: http://%s:%d", serveHost, servePort)
	log.Println()

	// 1. Load configuration from file
	cfg, err := loadConfig(serveConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Initialize agent from configuration
	agentInstance, err := createAgent(cfg, serveHost, servePort)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// 3. Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 4. Start HTTP server in goroutine
	errChan := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf("%s:%d", serveHost, servePort)
		if err := agentInstance.Start(addr); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("\n📥 Shutdown signal received, stopping agent...")
	case err := <-errChan:
		return err
	}

	// 5. Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := agentInstance.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent gracefully: %w", err)
	}

	log.Println("✅ Server stopped successfully")
	return nil
}

// loadConfig loads configuration from the specified file
func loadConfig(path string) (*config.Config, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("⚠️  Config file not found: %s, using defaults", path)
		return config.DefaultConfig(), nil
	}

	// Load from file
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	log.Printf("✅ Configuration loaded from %s", path)
	return cfg, nil
}

// createAgent creates and configures an agent from the configuration
func createAgent(cfg *config.Config, host string, port int) (*agent.AgentImpl, error) {
	log.Println("🔧 Initializing agent...")

	// Create agent builder
	b := builder.NewAgent(cfg.Agent.Name)

	// Configure LLM provider
	if err := configureLLM(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to configure LLM: %w", err)
	}

	// Configure storage
	if err := configureStorage(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to configure storage: %w", err)
	}

	// Configure protocol
	protocolMode := protocol.ProtocolAuto
	if cfg.Protocol.Mode != "" {
		switch cfg.Protocol.Mode {
		case "a2a":
			protocolMode = protocol.ProtocolA2A
		case "sage":
			protocolMode = protocol.ProtocolSAGE
		default:
			protocolMode = protocol.ProtocolAuto
		}
	}
	b.WithProtocol(protocolMode)

	// Configure A2A if enabled
	if cfg.A2A.Enabled {
		a2aConfig := &config.A2AConfig{
			Enabled:   true,
			Version:   cfg.A2A.Version,
			ServerURL: fmt.Sprintf("http://%s:%d/", host, port),
			Timeout:   cfg.A2A.Timeout,
		}
		b.WithA2AConfig(a2aConfig)
	}

	// Add lifecycle hooks
	b.BeforeStart(func(ctx context.Context) error {
		log.Printf("✅ Agent '%s' starting...", cfg.Agent.Name)
		log.Printf("🌐 Listening on http://%s:%d", host, port)
		log.Printf("📡 Protocol: %s", protocolMode)
		return nil
	})

	b.AfterStop(func(ctx context.Context) error {
		log.Printf("✅ Agent '%s' stopped", cfg.Agent.Name)
		return nil
	})

	// Set default message handler if not configured
	b.OnMessage(defaultMessageHandler())

	// Build agent
	agentInstance, err := b.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build agent: %w", err)
	}

	log.Println("✅ Agent initialized successfully")
	return agentInstance, nil
}

// configureLLM configures the LLM provider from config
func configureLLM(b *builder.Builder, cfg *config.Config) error {
	if cfg.LLM.Provider == "" {
		log.Println("⚠️  No LLM configured, agent will need custom message handler")
		return nil
	}

	var provider llm.Provider
	var err error

	switch cfg.LLM.Provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if cfg.LLM.APIKey != "" {
			apiKey = cfg.LLM.APIKey
		}
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key required (set OPENAI_API_KEY)")
		}
		provider = llm.OpenAI(&llm.OpenAIConfig{
			APIKey: apiKey,
			Model:  cfg.LLM.Model,
		})
		log.Printf("✅ LLM: OpenAI (%s)", cfg.LLM.Model)

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if cfg.LLM.APIKey != "" {
			apiKey = cfg.LLM.APIKey
		}
		if apiKey == "" {
			return fmt.Errorf("Anthropic API key required (set ANTHROPIC_API_KEY)")
		}
		provider = llm.Anthropic(&llm.AnthropicConfig{
			APIKey: apiKey,
			Model:  cfg.LLM.Model,
		})
		log.Printf("✅ LLM: Anthropic (%s)", cfg.LLM.Model)

	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if cfg.LLM.APIKey != "" {
			apiKey = cfg.LLM.APIKey
		}
		if apiKey == "" {
			return fmt.Errorf("Gemini API key required (set GEMINI_API_KEY)")
		}
		provider = llm.Gemini(&llm.GeminiConfig{
			APIKey: apiKey,
			Model:  cfg.LLM.Model,
		})
		log.Printf("✅ LLM: Gemini (%s)", cfg.LLM.Model)

	default:
		return fmt.Errorf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}

	if err != nil {
		return err
	}

	b.WithLLM(provider)
	return nil
}

// configureStorage configures the storage backend from config
func configureStorage(b *builder.Builder, cfg *config.Config) error {
	if cfg.Storage.Type == "" {
		log.Println("✅ Storage: Memory (default)")
		return nil
	}

	switch cfg.Storage.Type {
	case "memory":
		// Default, nothing to do
		log.Println("✅ Storage: Memory")

	case "redis":
		redisConfig := storage.DefaultRedisConfig()
		if cfg.Storage.Redis.Host != "" {
			redisConfig.Address = cfg.Storage.Redis.Host
			if cfg.Storage.Redis.Port > 0 {
				redisConfig.Address = fmt.Sprintf("%s:%d", cfg.Storage.Redis.Host, cfg.Storage.Redis.Port)
			}
			redisConfig.Password = cfg.Storage.Redis.Password
			redisConfig.DB = cfg.Storage.Redis.DB
		}
		redisStorage, err := storage.NewRedisStorage(redisConfig)
		if err != nil {
			return fmt.Errorf("failed to create Redis storage: %w", err)
		}
		b.WithStorage(redisStorage)
		log.Printf("✅ Storage: Redis (%s)", redisConfig.Address)

	case "postgres":
		pgConfig := storage.DefaultPostgresConfig()
		if cfg.Storage.Postgres.Host != "" {
			pgConfig.Host = cfg.Storage.Postgres.Host
		}
		if cfg.Storage.Postgres.Port > 0 {
			pgConfig.Port = cfg.Storage.Postgres.Port
		}
		if cfg.Storage.Postgres.User != "" {
			pgConfig.User = cfg.Storage.Postgres.User
		}
		if cfg.Storage.Postgres.Password != "" {
			pgConfig.Password = cfg.Storage.Postgres.Password
		}
		if cfg.Storage.Postgres.Database != "" {
			pgConfig.Database = cfg.Storage.Postgres.Database
		}
		if cfg.Storage.Postgres.SSLMode != "" {
			pgConfig.SSLMode = cfg.Storage.Postgres.SSLMode
		}
		pgStorage, err := storage.NewPostgresStorage(pgConfig)
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL storage: %w", err)
		}
		b.WithStorage(pgStorage)
		log.Printf("✅ Storage: PostgreSQL (%s:%d/%s)", pgConfig.Host, pgConfig.Port, pgConfig.Database)

	default:
		return fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}

	return nil
}

// defaultMessageHandler returns a basic message handler for testing
func defaultMessageHandler() agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		userText := msg.Text()
		if userText == "" {
			return fmt.Errorf("empty message received")
		}

		log.Printf("📨 Received message: %s", userText)

		// Echo the message back
		response := fmt.Sprintf("Echo: %s", userText)
		log.Printf("💬 Sending response: %s", response)

		return msg.Reply(response)
	}
}
