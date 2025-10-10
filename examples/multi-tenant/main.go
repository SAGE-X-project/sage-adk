// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

/*
Multi-Tenant Agent Example

This example demonstrates how to build a multi-tenant agent system with:
  - Tenant isolation
  - Per-tenant rate limiting
  - Per-tenant storage
  - Per-tenant configuration
  - Tenant authentication

Run:
	go run main.go
*/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sage-x-project/sage-adk/agent"
	"github.com/sage-x-project/sage-adk/cache"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage-adk/ratelimit"
	"github.com/sage-x-project/sage-adk/server"
	"github.com/sage-x-project/sage-adk/storage"
)

// TenantConfig holds configuration for a single tenant
type TenantConfig struct {
	ID          string
	Name        string
	RateLimit   int           // Requests per minute
	StorageType string        // "memory" or "redis"
	CacheSize   int           // Max cache entries
	Features    []string      // Enabled features
}

// TenantManager manages multiple tenants
type TenantManager struct {
	tenants   map[string]*Tenant
	storage   storage.Storage
	cache     cache.Cache
}

// Tenant represents a single tenant instance
type Tenant struct {
	config      TenantConfig
	agent       agent.Agent
	rateLimiter ratelimit.Limiter
	storage     storage.Storage
	cache       cache.Cache
	stats       *TenantStats
}

// TenantStats tracks tenant usage statistics
type TenantStats struct {
	RequestCount     int64
	AllowedRequests  int64
	DeniedRequests   int64
	AverageLatency   time.Duration
	TotalStorageUsed int64
}

func main() {
	fmt.Println("=== SAGE ADK Multi-Tenant Example ===\n")

	// Create tenant manager
	manager := NewTenantManager()

	// Register tenants with different configurations
	tenants := []TenantConfig{
		{
			ID:          "tenant-basic",
			Name:        "Basic Corp",
			RateLimit:   10,  // 10 req/min
			StorageType: "memory",
			CacheSize:   100,
			Features:    []string{"basic-chat"},
		},
		{
			ID:          "tenant-pro",
			Name:        "Pro Enterprise",
			RateLimit:   100, // 100 req/min
			StorageType: "memory",
			CacheSize:   1000,
			Features:    []string{"basic-chat", "advanced-analytics", "priority-support"},
		},
		{
			ID:          "tenant-enterprise",
			Name:        "Enterprise Solutions",
			RateLimit:   1000, // 1000 req/min
			StorageType: "memory",
			CacheSize:   10000,
			Features:    []string{"basic-chat", "advanced-analytics", "priority-support", "custom-models"},
		},
	}

	// Initialize all tenants
	for _, config := range tenants {
		if err := manager.RegisterTenant(config); err != nil {
			log.Fatalf("Failed to register tenant %s: %v", config.ID, err)
		}
		fmt.Printf("✓ Registered tenant: %s (%s)\n", config.Name, config.ID)
		fmt.Printf("  Rate Limit: %d req/min\n", config.RateLimit)
		fmt.Printf("  Features: %v\n\n", config.Features)
	}

	// Start HTTP server with tenant routing
	fmt.Println("Starting multi-tenant server on :8080")
	fmt.Println("Tenants:")
	fmt.Println("  - Basic Corp:       http://localhost:8080/tenant-basic")
	fmt.Println("  - Pro Enterprise:   http://localhost:8080/tenant-pro")
	fmt.Println("  - Enterprise:       http://localhost:8080/tenant-enterprise")
	fmt.Println()

	// Run simulation
	go runSimulation(manager)

	// Start server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Tenant stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := manager.GetAllStats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{\n")
		for id, stat := range stats {
			fmt.Fprintf(w, "  \"%s\": {\n", id)
			fmt.Fprintf(w, "    \"requests\": %d,\n", stat.RequestCount)
			fmt.Fprintf(w, "    \"allowed\": %d,\n", stat.AllowedRequests)
			fmt.Fprintf(w, "    \"denied\": %d,\n", stat.DeniedRequests)
			fmt.Fprintf(w, "    \"avg_latency_ms\": %d\n", stat.AverageLatency.Milliseconds())
			fmt.Fprintf(w, "  },\n")
		}
		fmt.Fprintf(w, "}\n")
	})

	// Tenant message endpoints
	for tenantID := range manager.tenants {
		id := tenantID // Capture for closure
		mux.HandleFunc("/"+id, func(w http.ResponseWriter, r *http.Request) {
			manager.HandleTenantRequest(w, r, id)
		})
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// NewTenantManager creates a new tenant manager
func NewTenantManager() *TenantManager {
	return &TenantManager{
		tenants: make(map[string]*Tenant),
		storage: storage.NewMemoryStorage(),
		cache:   cache.NewMemoryCache(cache.DefaultCacheConfig()),
	}
}

// RegisterTenant registers a new tenant
func (tm *TenantManager) RegisterTenant(config TenantConfig) error {
	// Create tenant-specific storage
	var tenantStorage storage.Storage
	switch config.StorageType {
	case "memory":
		tenantStorage = storage.NewMemoryStorage()
	default:
		tenantStorage = storage.NewMemoryStorage()
	}

	// Create tenant-specific cache
	tenantCache := cache.NewMemoryCache(cache.CacheConfig{
		MaxSize:        config.CacheSize,
		DefaultTTL:     5 * time.Minute,
		EvictionPolicy: cache.EvictionPolicyLRU,
		EnableMetrics:  true,
	})

	// Create rate limiter for tenant
	rateLimiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Rate:     float64(config.RateLimit) / 60.0, // Convert per minute to per second
		Capacity: config.RateLimit,
	})

	// Create agent with tenant-specific handler
	agentHandler := createTenantHandler(config)

	agentImpl := agent.NewAgent(agent.AgentConfig{
		Name:        config.Name,
		Description: fmt.Sprintf("Agent for %s", config.Name),
		Version:     "1.0.0",
	})

	agentImpl.SetHandler(agentHandler)

	// Add tenant-specific middleware
	agentImpl.UseMiddleware(createRateLimitMiddleware(rateLimiter))
	agentImpl.UseMiddleware(createTenantLoggingMiddleware(config.ID))

	tenant := &Tenant{
		config:      config,
		agent:       agentImpl,
		rateLimiter: rateLimiter,
		storage:     tenantStorage,
		cache:       tenantCache,
		stats:       &TenantStats{},
	}

	tm.tenants[config.ID] = tenant
	return nil
}

// HandleTenantRequest handles HTTP requests for a specific tenant
func (tm *TenantManager) HandleTenantRequest(w http.ResponseWriter, r *http.Request, tenantID string) {
	tenant, exists := tm.tenants[tenantID]
	if !exists {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	// Create message from request
	ctx := context.Background()
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{
			types.NewTextPart(r.URL.Query().Get("q")),
		},
	)
	msg.Metadata = map[string]interface{}{
		"tenant_id": tenantID,
	}

	// Track request
	start := time.Now()
	tenant.stats.RequestCount++

	// Process message
	response, err := tenant.agent.Process(ctx, msg)
	if err != nil {
		tenant.stats.DeniedRequests++
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	tenant.stats.AllowedRequests++

	// Update average latency
	latency := time.Since(start)
	tenant.stats.AverageLatency =
		(tenant.stats.AverageLatency*time.Duration(tenant.stats.AllowedRequests-1) + latency) /
		time.Duration(tenant.stats.AllowedRequests)

	// Return response
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Tenant-ID", tenantID)
	w.Header().Set("X-Processing-Time-Ms", fmt.Sprintf("%d", latency.Milliseconds()))

	if len(response.Parts) > 0 {
		if textPart, ok := response.Parts[0].(*types.TextPart); ok {
			fmt.Fprint(w, textPart.Text)
		}
	}
}

// GetAllStats returns stats for all tenants
func (tm *TenantManager) GetAllStats() map[string]*TenantStats {
	stats := make(map[string]*TenantStats)
	for id, tenant := range tm.tenants {
		stats[id] = tenant.stats
	}
	return stats
}

// createTenantHandler creates a tenant-specific message handler
func createTenantHandler(config TenantConfig) agent.MessageHandler {
	return func(ctx context.Context, msgCtx agent.MessageContext) error {
		text := msgCtx.Text()

		// Check feature availability
		response := fmt.Sprintf("[%s] Received: %s\n", config.Name, text)
		response += fmt.Sprintf("Available features: %v\n", config.Features)
		response += fmt.Sprintf("Processing with %s tier configuration", getTier(config))

		return msgCtx.Reply(types.NewMessage(
			types.MessageRoleAssistant,
			[]types.Part{types.NewTextPart(response)},
		))
	}
}

// createRateLimitMiddleware creates rate limiting middleware
func createRateLimitMiddleware(limiter ratelimit.Limiter) func(agent.MessageHandler) agent.MessageHandler {
	return func(next agent.MessageHandler) agent.MessageHandler {
		return func(ctx context.Context, msgCtx agent.MessageContext) error {
			// Extract tenant ID
			tenantID := "default"
			if metadata := msgCtx.Message().Metadata; metadata != nil {
				if id, ok := metadata["tenant_id"].(string); ok {
					tenantID = id
				}
			}

			// Check rate limit
			if !limiter.Allow(tenantID) {
				return fmt.Errorf("rate limit exceeded for tenant %s", tenantID)
			}

			return next(ctx, msgCtx)
		}
	}
}

// createTenantLoggingMiddleware creates logging middleware
func createTenantLoggingMiddleware(tenantID string) func(agent.MessageHandler) agent.MessageHandler {
	return func(next agent.MessageHandler) agent.MessageHandler {
		return func(ctx context.Context, msgCtx agent.MessageContext) error {
			start := time.Now()
			log.Printf("[%s] Processing message: %s", tenantID, msgCtx.Message().MessageID)

			err := next(ctx, msgCtx)

			duration := time.Since(start)
			if err != nil {
				log.Printf("[%s] Error after %v: %v", tenantID, duration, err)
			} else {
				log.Printf("[%s] Completed in %v", tenantID, duration)
			}

			return err
		}
	}
}

// getTier returns tier name based on configuration
func getTier(config TenantConfig) string {
	if config.RateLimit >= 1000 {
		return "Enterprise"
	} else if config.RateLimit >= 100 {
		return "Professional"
	}
	return "Basic"
}

// runSimulation runs a simulation of tenant requests
func runSimulation(manager *TenantManager) {
	time.Sleep(2 * time.Second)

	fmt.Println("\n=== Running Simulation ===\n")

	// Simulate requests from different tenants
	scenarios := []struct {
		tenantID string
		requests int
		query    string
	}{
		{"tenant-basic", 15, "Hello from Basic Corp"},
		{"tenant-pro", 50, "Hello from Pro Enterprise"},
		{"tenant-enterprise", 100, "Hello from Enterprise Solutions"},
	}

	for _, scenario := range scenarios {
		fmt.Printf("Simulating %d requests from %s\n", scenario.requests, scenario.tenantID)

		allowed := 0
		denied := 0

		for i := 0; i < scenario.requests; i++ {
			tenant := manager.tenants[scenario.tenantID]
			if tenant.rateLimiter.Allow(scenario.tenantID) {
				allowed++
			} else {
				denied++
			}
			time.Sleep(50 * time.Millisecond) // 20 req/sec
		}

		fmt.Printf("  ✓ Allowed: %d\n", allowed)
		fmt.Printf("  ✗ Denied: %d\n", denied)
		fmt.Printf("  Rate Limit: %d req/min\n\n", tenant.config.RateLimit)
	}

	// Print final stats
	fmt.Println("=== Final Statistics ===\n")
	stats := manager.GetAllStats()
	for id, stat := range stats {
		tenant := manager.tenants[id]
		fmt.Printf("%s (%s):\n", tenant.config.Name, id)
		fmt.Printf("  Total Requests: %d\n", stat.RequestCount)
		fmt.Printf("  Allowed: %d\n", stat.AllowedRequests)
		fmt.Printf("  Denied: %d\n", stat.DeniedRequests)
		if stat.AllowedRequests > 0 {
			fmt.Printf("  Avg Latency: %v\n", stat.AverageLatency)
		}
		fmt.Println()
	}
}
