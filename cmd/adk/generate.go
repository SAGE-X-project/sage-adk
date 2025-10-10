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

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate [type] [name]",
	Short: "Generate code for SAGE ADK components",
	Long: `Generate boilerplate code for various SAGE ADK components.

Supported types:
  provider    - Generate a new LLM provider
  middleware  - Generate a new middleware
  adapter     - Generate a new protocol adapter

Example:
  adk generate provider my-llm
  adk generate middleware auth
  adk generate adapter my-protocol`,
	Args: cobra.ExactArgs(2),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	genType := args[0]
	name := args[1]

	switch genType {
	case "provider":
		return generateProvider(name)
	case "middleware":
		return generateMiddleware(name)
	case "adapter":
		return generateAdapter(name)
	default:
		return fmt.Errorf("unknown generate type: %s (supported: provider, middleware, adapter)", genType)
	}
}

func generateProvider(name string) error {
	filename := fmt.Sprintf("%s_provider.go", name)
	content := fmt.Sprintf(`package llm

import (
	"context"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// %sProvider is a custom LLM provider.
type %sProvider struct {
	apiKey string
	config *%sConfig
}

// %sConfig contains configuration for %s provider.
type %sConfig struct {
	APIKey string
	Model  string
}

// New%s creates a new %s provider.
func New%s(cfg *%sConfig) (*%sProvider, error) {
	return &%sProvider{
		apiKey: cfg.APIKey,
		config: cfg,
	}, nil
}

// Name returns the provider name.
func (p *%sProvider) Name() string {
	return "%s"
}

// Generate generates a response from the LLM.
func (p *%sProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	// TODO: Implement your LLM provider logic here
	return nil, nil
}
`, name, name, name, name, name, name, name, name, name, name, name, name, name, name, name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create provider file: %w", err)
	}

	fmt.Printf("✓ Generated provider: %s\n", filename)
	return nil
}

func generateMiddleware(name string) error {
	filename := fmt.Sprintf("%s_middleware.go", name)
	content := fmt.Sprintf(`package middleware

import (
	"context"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// %s is a custom middleware.
func %s() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// TODO: Pre-processing logic here

			// Call next handler
			response, err := next(ctx, msg)

			// TODO: Post-processing logic here

			return response, err
		}
	}
}
`, name, name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create middleware file: %w", err)
	}

	fmt.Printf("✓ Generated middleware: %s\n", filename)
	return nil
}

func generateAdapter(name string) error {
	dir := filepath.Join("adapters", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create adapter directory: %w", err)
	}

	// Generate adapter.go
	adapterFile := filepath.Join(dir, "adapter.go")
	content := fmt.Sprintf(`package %s

import (
	"context"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Adapter implements the %s protocol.
type Adapter struct {
	config *Config
}

// Config contains configuration for %s adapter.
type Config struct {
	// Add your configuration fields here
}

// NewAdapter creates a new %s adapter.
func NewAdapter(cfg *Config) (*Adapter, error) {
	return &Adapter{
		config: cfg,
	}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "%s"
}

// SendMessage sends a message using this protocol.
func (a *Adapter) SendMessage(ctx context.Context, msg *types.Message) error {
	// TODO: Implement send logic
	return nil
}

// ReceiveMessage receives a message using this protocol.
func (a *Adapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
	// TODO: Implement receive logic
	return nil, nil
}

// Verify verifies a message according to this protocol's requirements.
func (a *Adapter) Verify(ctx context.Context, msg *types.Message) error {
	// TODO: Implement verification logic
	return nil
}

// SupportsStreaming returns whether this protocol supports streaming.
func (a *Adapter) SupportsStreaming() bool {
	return false // Change to true if streaming is supported
}

// Stream sends a message and streams the response.
func (a *Adapter) Stream(ctx context.Context, fn protocol.StreamFunc) error {
	// TODO: Implement streaming logic
	return nil
}
`, name, name, name, name, name)

	if err := os.WriteFile(adapterFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create adapter file: %w", err)
	}

	fmt.Printf("✓ Generated adapter: %s\n", adapterFile)
	return nil
}
