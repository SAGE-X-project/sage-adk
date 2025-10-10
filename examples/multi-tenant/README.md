# Multi-Tenant Agent Example

This example demonstrates how to build a multi-tenant agent system with complete tenant isolation and per-tenant configurations.

## Features

### 1. Tenant Isolation
- Separate storage per tenant
- Isolated cache per tenant
- Independent configurations
- Privacy and data segregation

### 2. Per-Tenant Rate Limiting
- Different rate limits per tier
- Token bucket algorithm
- Burst handling
- Fair resource allocation

### 3. Tiered Service Plans
- **Basic**: 10 requests/minute, basic features
- **Professional**: 100 requests/minute, advanced features
- **Enterprise**: 1000 requests/minute, all features

### 4. Tenant Management
- Dynamic tenant registration
- Configuration management
- Usage statistics
- Resource monitoring

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              HTTP Server (:8080)                     │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│              Tenant Manager                          │
│  - Tenant routing                                    │
│  - Authentication                                    │
│  - Statistics aggregation                            │
└─────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Tenant 1   │  │   Tenant 2   │  │   Tenant 3   │
├──────────────┤  ├──────────────┤  ├──────────────┤
│ Agent        │  │ Agent        │  │ Agent        │
│ Rate Limiter │  │ Rate Limiter │  │ Rate Limiter │
│ Storage      │  │ Storage      │  │ Storage      │
│ Cache        │  │ Cache        │  │ Cache        │
└──────────────┘  └──────────────┘  └──────────────┘
```

## Usage

### Starting the Server

```bash
go run main.go
```

The server will start on port 8080 with three pre-configured tenants:
- Basic Corp: `http://localhost:8080/tenant-basic`
- Pro Enterprise: `http://localhost:8080/tenant-pro`
- Enterprise Solutions: `http://localhost:8080/tenant-enterprise`

### Making Requests

```bash
# Basic tier tenant
curl "http://localhost:8080/tenant-basic?q=Hello"

# Professional tier tenant
curl "http://localhost:8080/tenant-pro?q=Analyze+data"

# Enterprise tier tenant
curl "http://localhost:8080/tenant-enterprise?q=Custom+model+query"
```

### Viewing Statistics

```bash
curl http://localhost:8080/stats
```

Returns per-tenant statistics:
```json
{
  "tenant-basic": {
    "requests": 150,
    "allowed": 100,
    "denied": 50,
    "avg_latency_ms": 23
  },
  "tenant-pro": {
    "requests": 500,
    "allowed": 480,
    "denied": 20,
    "avg_latency_ms": 18
  },
  "tenant-enterprise": {
    "requests": 2000,
    "allowed": 2000,
    "denied": 0,
    "avg_latency_ms": 15
  }
}
```

## Configuration

### Tenant Configuration

```go
type TenantConfig struct {
    ID          string      // Unique tenant identifier
    Name        string      // Display name
    RateLimit   int         // Requests per minute
    StorageType string      // "memory" or "redis"
    CacheSize   int         // Max cache entries
    Features    []string    // Enabled features
}
```

### Example Configurations

#### Basic Tier
```go
TenantConfig{
    ID:          "tenant-basic",
    Name:        "Basic Corp",
    RateLimit:   10,        // 10 req/min
    StorageType: "memory",
    CacheSize:   100,
    Features:    []string{"basic-chat"},
}
```

#### Professional Tier
```go
TenantConfig{
    ID:          "tenant-pro",
    Name:        "Pro Enterprise",
    RateLimit:   100,       // 100 req/min
    StorageType: "memory",
    CacheSize:   1000,
    Features:    []string{
        "basic-chat",
        "advanced-analytics",
        "priority-support",
    },
}
```

#### Enterprise Tier
```go
TenantConfig{
    ID:          "tenant-enterprise",
    Name:        "Enterprise Solutions",
    RateLimit:   1000,      // 1000 req/min
    StorageType: "redis",   // Persistent storage
    CacheSize:   10000,
    Features:    []string{
        "basic-chat",
        "advanced-analytics",
        "priority-support",
        "custom-models",
        "dedicated-resources",
    },
}
```

## Tenant Isolation

### Storage Isolation
Each tenant has its own storage instance:
```go
tenantStorage := storage.NewMemoryStorage()
// or
tenantStorage := storage.NewRedisStorage(redis.Options{
    DB: tenantID, // Separate Redis DB per tenant
})
```

### Cache Isolation
Each tenant has its own cache:
```go
tenantCache := cache.NewMemoryCache(cache.CacheConfig{
    MaxSize:    config.CacheSize,
    DefaultTTL: 5 * time.Minute,
})
```

### Rate Limit Isolation
Each tenant has its own rate limiter:
```go
rateLimiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
    Rate:     float64(config.RateLimit) / 60.0,
    Capacity: config.RateLimit,
})
```

## Middleware Chain

Each tenant agent uses a middleware chain:

```go
// 1. Rate limiting
agentImpl.UseMiddleware(createRateLimitMiddleware(rateLimiter))

// 2. Logging
agentImpl.UseMiddleware(createTenantLoggingMiddleware(tenantID))

// 3. Metrics
agentImpl.UseMiddleware(createMetricsMiddleware(tenantID))

// 4. Authentication
agentImpl.UseMiddleware(createAuthMiddleware(tenantID))
```

## Monitoring

### Per-Tenant Metrics
```go
type TenantStats struct {
    RequestCount     int64         // Total requests
    AllowedRequests  int64         // Allowed by rate limiter
    DeniedRequests   int64         // Denied by rate limiter
    AverageLatency   time.Duration // Average processing time
    TotalStorageUsed int64         // Storage usage in bytes
}
```

### Dashboard Integration
Expose metrics for Prometheus:
```go
// Custom metrics per tenant
tenantRequestsTotal := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "tenant_requests_total",
        Help: "Total requests per tenant",
    },
    []string{"tenant_id", "status"},
)
```

## Best Practices

### 1. Resource Limits
```go
// Set reasonable limits per tier
config := TenantConfig{
    RateLimit:   100,        // Prevent abuse
    CacheSize:   1000,       // Limit memory usage
    MaxStorage:  1000000000, // 1GB storage limit
}
```

### 2. Fair Scheduling
```go
// Use token bucket for burst tolerance
rateLimiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
    Rate:     float64(rateLimit) / 60.0, // Smooth rate
    Capacity: rateLimit * 2,              // Allow 2x burst
})
```

### 3. Data Isolation
```go
// Use separate storage per tenant
switch config.StorageType {
case "redis":
    storage := redis.NewClient(&redis.Options{
        DB: tenantDB, // Separate DB
    })
case "postgres":
    storage := postgres.New(fmt.Sprintf(
        "schema=%s", tenantID, // Separate schema
    ))
}
```

### 4. Security
```go
// Validate tenant ID in all requests
middleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantID(r)
        if !isValidTenant(tenantID) {
            http.Error(w, "Invalid tenant", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## Scaling

### Horizontal Scaling
```go
// Use distributed rate limiter for multi-instance deployment
limiter := ratelimit.NewDistributed(ratelimit.DistributedConfig{
    RedisClient: redisClient,
    KeyPrefix:   fmt.Sprintf("tenant:%s:", tenantID),
    Limit:       config.RateLimit,
    Window:      time.Minute,
})
```

### Database Sharding
```go
// Shard tenants across multiple databases
func getStorageForTenant(tenantID string) storage.Storage {
    shard := hashTenantID(tenantID) % numShards
    return storageShards[shard]
}
```

## Testing

```bash
# Run load test for specific tenant
hey -n 1000 -c 10 -q 20 "http://localhost:8080/tenant-pro?q=test"

# Monitor rate limiting
watch -n 1 'curl -s http://localhost:8080/stats | jq'
```

## Production Deployment

### With Docker Compose

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis:6379
      - POSTGRES_URL=postgres://db:5432
    depends_on:
      - redis
      - postgres

  redis:
    image: redis:alpine
    volumes:
      - redis-data:/data

  postgres:
    image: postgres:15
    volumes:
      - pg-data:/var/lib/postgresql/data
```

### With Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: multi-tenant-agent
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: agent
        image: sage-adk-multi-tenant:latest
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
          requests:
            memory: "256Mi"
            cpu: "250m"
```

## See Also

- [Rate Limiting Example](../rate-limiting/)
- [Caching Example](../caching/)
- [Distributed Tracing](../distributed-tracing/)
