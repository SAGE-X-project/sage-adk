# PRD: PHASE 3 - Production-Ready Features

**Version:** 1.0
**Date:** 2025-10-08
**Status:** Draft
**Target Release:** v0.2.0

## Executive Summary

This PRD outlines the remaining features required to make SAGE ADK production-ready. These features focus on observability, reliability, performance, and operational excellence for AI agent deployments.

## Table of Contents

1. [Monitoring & Observability](#1-monitoring--observability)
2. [Rate Limiting](#2-rate-limiting)
3. [Advanced Error Recovery](#3-advanced-error-recovery)
4. [Performance Tuning](#4-performance-tuning)
5. [Production Deployment Guide](#5-production-deployment-guide)

---

## 1. Monitoring & Observability

### 1.1 Overview

Enable comprehensive monitoring, metrics collection, and distributed tracing for AI agents in production environments.

### 1.2 Business Goals

- **Visibility**: Full observability into agent behavior and performance
- **Debugging**: Quick identification and resolution of issues
- **Optimization**: Data-driven performance improvements
- **Compliance**: Audit trail and compliance reporting

### 1.3 User Stories

**As a DevOps Engineer:**
- I want to monitor agent health in real-time so I can detect issues proactively
- I want to track LLM API usage and costs so I can optimize spending
- I want to set up alerts for anomalies so I can respond quickly to incidents

**As a Developer:**
- I want detailed error logs with context so I can debug issues efficiently
- I want to trace requests across distributed components so I can identify bottlenecks
- I want to profile agent performance so I can optimize critical paths

**As a Product Manager:**
- I want usage analytics so I can understand user engagement
- I want cost metrics so I can forecast infrastructure expenses
- I want SLA compliance reports so I can measure service quality

### 1.4 Requirements

#### 1.4.1 Metrics (Must Have)

**System Metrics:**
- Agent health status (up/down, ready/not ready)
- Request rate (requests per second)
- Error rate and error types
- Response latency (p50, p95, p99)
- Active goroutines and memory usage

**LLM Metrics:**
- LLM API calls (count, rate)
- Token usage (prompt, completion, total)
- LLM response latency
- LLM error rate by provider
- Cost per request (estimated)

**Protocol Metrics:**
- A2A message count and latency
- SAGE handshake success/failure rate
- DID resolution time
- Message signing/verification time

**Storage Metrics:**
- Storage operations (get/set/delete/list)
- Storage latency by operation
- Cache hit/miss ratio (Redis)
- Connection pool usage

**Middleware Metrics:**
- Middleware execution time by name
- Pre/post processing latency
- Middleware error rate

**Resilience Metrics:**
- Retry attempts and success rate
- Circuit breaker state changes
- Bulkhead queue depth and rejections
- Timeout occurrences

#### 1.4.2 Logging (Must Have)

**Structured Logging:**
- JSON format for machine parsing
- Log levels: DEBUG, INFO, WARN, ERROR, FATAL
- Context propagation (request ID, trace ID, agent ID)
- Correlation IDs across distributed calls

**Log Categories:**
- Agent lifecycle (start, stop, config changes)
- Message handling (received, processed, replied)
- LLM interactions (request, response, errors)
- Protocol operations (handshake, signing, verification)
- Storage operations (with query details)
- Security events (auth failures, signature mismatches)

**Log Sampling:**
- Sample DEBUG logs in production (1-10%)
- Always log WARN and above
- Configurable sampling rate

#### 1.4.3 Tracing (Should Have)

**Distributed Tracing:**
- OpenTelemetry integration
- Trace propagation across services
- Span creation for:
  - Message handling
  - LLM API calls
  - Storage operations
  - Protocol operations
  - Middleware execution

**Trace Attributes:**
- Agent ID, message ID, session ID
- User ID (if available)
- Protocol type (A2A/SAGE)
- LLM provider and model
- Error details

#### 1.4.4 Health Checks (Must Have)

**Liveness Probe:**
- `/health/live` endpoint
- Returns 200 if agent is running
- No external dependency checks

**Readiness Probe:**
- `/health/ready` endpoint
- Returns 200 if agent is ready to serve traffic
- Checks:
  - LLM provider connectivity
  - Storage backend connectivity
  - Protocol adapter status

**Startup Probe:**
- `/health/startup` endpoint
- Returns 200 when initialization complete
- Used for slow-starting agents

### 1.5 Technical Design

#### 1.5.1 Architecture

```
┌─────────────────────────────────────────────┐
│                 Agent Core                   │
│  ┌─────────────────────────────────────┐   │
│  │      Metrics Collector              │   │
│  │  - Prometheus Registry              │   │
│  │  - Custom Metrics                   │   │
│  └─────────────────────────────────────┘   │
│  ┌─────────────────────────────────────┐   │
│  │      Logger                          │   │
│  │  - Structured (JSON)                │   │
│  │  - Context-aware                    │   │
│  │  - Sampling                         │   │
│  └─────────────────────────────────────┘   │
│  ┌─────────────────────────────────────┐   │
│  │      Tracer (OpenTelemetry)         │   │
│  │  - Span creation                    │   │
│  │  - Context propagation              │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
           │         │         │
           ▼         ▼         ▼
    Prometheus   Loki/ELK   Jaeger/Tempo
```

#### 1.5.2 Package Structure

```
sage-adk/
├── observability/
│   ├── doc.go
│   ├── metrics/
│   │   ├── collector.go       # Metrics collection
│   │   ├── prometheus.go      # Prometheus exporter
│   │   ├── agent.go           # Agent metrics
│   │   ├── llm.go             # LLM metrics
│   │   ├── protocol.go        # Protocol metrics
│   │   ├── storage.go         # Storage metrics
│   │   └── middleware.go      # Middleware metrics
│   ├── logging/
│   │   ├── logger.go          # Logger interface
│   │   ├── structured.go      # JSON logger
│   │   ├── context.go         # Context propagation
│   │   ├── sampling.go        # Log sampling
│   │   └── adapters/          # Log adapters (zap, zerolog)
│   ├── tracing/
│   │   ├── tracer.go          # Tracer interface
│   │   ├── otel.go            # OpenTelemetry
│   │   ├── context.go         # Trace context
│   │   └── spans.go           # Span helpers
│   └── health/
│       ├── checker.go         # Health check interface
│       ├── liveness.go        # Liveness probe
│       ├── readiness.go       # Readiness probe
│       └── startup.go         # Startup probe
```

#### 1.5.3 Configuration

```go
type ObservabilityConfig struct {
    // Metrics
    Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

    // Logging
    Logging LoggingConfig `yaml:"logging" json:"logging"`

    // Tracing
    Tracing TracingConfig `yaml:"tracing" json:"tracing"`

    // Health checks
    Health HealthConfig `yaml:"health" json:"health"`
}

type MetricsConfig struct {
    Enabled  bool   `yaml:"enabled" json:"enabled"`
    Port     int    `yaml:"port" json:"port"`                // Prometheus port
    Path     string `yaml:"path" json:"path"`                // Metrics endpoint
    Interval int    `yaml:"interval" json:"interval"`        // Scrape interval (seconds)
}

type LoggingConfig struct {
    Level       string  `yaml:"level" json:"level"`          // DEBUG, INFO, WARN, ERROR
    Format      string  `yaml:"format" json:"format"`        // json, text
    Output      string  `yaml:"output" json:"output"`        // stdout, file
    FilePath    string  `yaml:"file_path" json:"file_path"`
    SamplingRate float64 `yaml:"sampling_rate" json:"sampling_rate"` // 0.0-1.0
}

type TracingConfig struct {
    Enabled     bool    `yaml:"enabled" json:"enabled"`
    Endpoint    string  `yaml:"endpoint" json:"endpoint"`    // OTLP endpoint
    ServiceName string  `yaml:"service_name" json:"service_name"`
    SamplingRate float64 `yaml:"sampling_rate" json:"sampling_rate"` // 0.0-1.0
}

type HealthConfig struct {
    Enabled       bool   `yaml:"enabled" json:"enabled"`
    Port          int    `yaml:"port" json:"port"`
    LivenessPath  string `yaml:"liveness_path" json:"liveness_path"`
    ReadinessPath string `yaml:"readiness_path" json:"readiness_path"`
    StartupPath   string `yaml:"startup_path" json:"startup_path"`
}
```

### 1.6 Success Metrics

- **Coverage**: 100% of critical paths instrumented
- **Performance**: <1ms overhead for metrics collection
- **Reliability**: 99.9% metrics delivery rate
- **Adoption**: All examples include observability

---

## 2. Rate Limiting

### 2.1 Overview

Protect AI agents from abuse, control costs, and ensure fair resource usage through configurable rate limiting.

### 2.2 Business Goals

- **Cost Control**: Prevent runaway LLM API costs
- **Abuse Prevention**: Protect against malicious or accidental overuse
- **Fair Usage**: Ensure equitable resource distribution
- **SLA Compliance**: Meet provider rate limits

### 2.3 User Stories

**As a Platform Owner:**
- I want to limit API calls per user so I can control costs
- I want to prevent DDoS attacks so my service stays available
- I want to enforce tier-based limits so I can monetize appropriately

**As a Developer:**
- I want configurable rate limits so I can adjust based on use case
- I want clear error messages when limits are exceeded so users understand
- I want rate limit metrics so I can optimize configurations

### 2.4 Requirements

#### 2.4.1 Rate Limit Types (Must Have)

**Request Rate Limiting:**
- Requests per second (RPS)
- Requests per minute (RPM)
- Requests per hour (RPH)
- Requests per day (RPD)

**Token Rate Limiting:**
- Tokens per second (TPS)
- Tokens per minute (TPM)
- Tokens per hour (TPH)
- Tokens per day (TPD)

**Concurrent Request Limiting:**
- Max concurrent requests per user
- Max concurrent requests globally

**Cost-Based Limiting:**
- Max cost per time period
- Budget exhaustion handling

#### 2.4.2 Granularity (Must Have)

**Global Rate Limits:**
- Apply to all requests
- Shared across all users

**Per-User Rate Limits:**
- Unique limits per user ID
- User identification via API key, DID, or custom ID

**Per-Endpoint Rate Limits:**
- Different limits for different operations
- Message handling vs. admin operations

**Per-LLM-Provider Limits:**
- Respect provider-specific limits
- OpenAI: 3,500 RPM (GPT-3.5-turbo)
- Anthropic: 5 requests/min (free tier)
- Gemini: 60 RPM (free tier)

#### 2.4.3 Strategies (Must Have)

**Token Bucket:**
- Refill rate configurable
- Burst capacity configurable
- Best for smooth traffic

**Fixed Window:**
- Simple implementation
- Risk of boundary spikes
- Good for daily/hourly limits

**Sliding Window:**
- More accurate than fixed window
- Higher memory usage
- Best for precise control

**Leaky Bucket:**
- Smooth output rate
- Queue-based
- Good for backend protection

#### 2.4.4 Storage Backends (Must Have)

**In-Memory:**
- Fast, no external deps
- Lost on restart
- Single instance only

**Redis:**
- Distributed rate limiting
- Persistent across restarts
- Supports all strategies

**PostgreSQL:**
- Persistent, queryable
- Higher latency
- Good for audit trails

#### 2.4.5 Response Handling (Must Have)

**Headers:**
- `X-RateLimit-Limit`: Total limit
- `X-RateLimit-Remaining`: Remaining quota
- `X-RateLimit-Reset`: Reset timestamp
- `Retry-After`: Seconds until retry allowed

**Error Response:**
```json
{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Try again in 42 seconds.",
  "retry_after": 42,
  "limit": 100,
  "remaining": 0,
  "reset": 1696809600
}
```

**HTTP Status:**
- `429 Too Many Requests`

### 2.5 Technical Design

#### 2.5.1 Architecture

```
Request → Rate Limiter → Agent Handler
             ↓
          Storage
      (Memory/Redis/PG)
```

#### 2.5.2 Package Structure

```
sage-adk/
├── ratelimit/
│   ├── doc.go
│   ├── limiter.go           # Rate limiter interface
│   ├── types.go             # Config types
│   ├── strategies/
│   │   ├── token_bucket.go
│   │   ├── fixed_window.go
│   │   ├── sliding_window.go
│   │   └── leaky_bucket.go
│   ├── storage/
│   │   ├── memory.go
│   │   ├── redis.go
│   │   └── postgres.go
│   ├── middleware.go        # Rate limit middleware
│   └── errors.go
```

#### 2.5.3 Configuration

```go
type RateLimitConfig struct {
    Enabled  bool              `yaml:"enabled" json:"enabled"`
    Strategy string            `yaml:"strategy" json:"strategy"` // token_bucket, fixed_window, sliding_window
    Storage  string            `yaml:"storage" json:"storage"`   // memory, redis, postgres

    // Global limits
    Global RateLimitRule `yaml:"global" json:"global"`

    // Per-user limits
    PerUser RateLimitRule `yaml:"per_user" json:"per_user"`

    // LLM provider limits
    LLMProviders map[string]RateLimitRule `yaml:"llm_providers" json:"llm_providers"`
}

type RateLimitRule struct {
    // Request limits
    RequestsPerSecond int `yaml:"requests_per_second" json:"requests_per_second"`
    RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
    RequestsPerHour   int `yaml:"requests_per_hour" json:"requests_per_hour"`
    RequestsPerDay    int `yaml:"requests_per_day" json:"requests_per_day"`

    // Token limits (for LLM)
    TokensPerSecond   int `yaml:"tokens_per_second" json:"tokens_per_second"`
    TokensPerMinute   int `yaml:"tokens_per_minute" json:"tokens_per_minute"`
    TokensPerHour     int `yaml:"tokens_per_hour" json:"tokens_per_hour"`
    TokensPerDay      int `yaml:"tokens_per_day" json:"tokens_per_day"`

    // Concurrency
    MaxConcurrent     int `yaml:"max_concurrent" json:"max_concurrent"`

    // Cost (future)
    MaxCostPerHour    float64 `yaml:"max_cost_per_hour" json:"max_cost_per_hour"`
}
```

### 2.6 Success Metrics

- **Protection**: 0 cost overruns due to abuse
- **Fairness**: Gini coefficient < 0.3 for resource distribution
- **Performance**: <100μs rate limit check latency
- **Accuracy**: <1% error rate in limit enforcement

---

## 3. Advanced Error Recovery

### 3.1 Overview

Enhance error handling with automatic recovery, fallback strategies, and graceful degradation.

### 3.2 Business Goals

- **Reliability**: Minimize user-facing errors
- **Resilience**: Automatic recovery from transient failures
- **User Experience**: Graceful degradation instead of hard failures
- **Operational Cost**: Reduce manual intervention

### 3.3 User Stories

**As an End User:**
- I want the agent to retry failed requests so I don't have to
- I want fallback responses when the LLM is unavailable so I still get help
- I want clear error messages so I know what went wrong

**As a Developer:**
- I want automatic LLM fallback so my agent stays available
- I want dead letter queues for failed messages so I can debug later
- I want error categorization so I can handle different errors appropriately

### 3.4 Requirements

#### 3.4.1 Retry Enhancements (Must Have)

**Already Implemented:**
- Exponential backoff
- Max attempts
- Retry conditions

**Enhancements Needed:**
- **Jitter**: Randomized backoff to prevent thundering herd
- **Backoff Multiplier**: Configurable growth rate
- **Per-Error-Type Retry**: Different strategies for different errors
- **Retry Budget**: Limit retries across time window

#### 3.4.2 Fallback Strategies (Must Have)

**LLM Fallbacks:**
- Primary: GPT-4 → Fallback: GPT-3.5-turbo → Fallback: Claude
- Cost-aware fallbacks (expensive → cheap)
- Quality-aware fallbacks (high quality → lower quality)

**Response Fallbacks:**
- Primary: LLM → Fallback: Cached response → Fallback: Default message
- Pattern matching for common queries
- Static responses for critical flows

**Storage Fallbacks:**
- Primary: PostgreSQL → Fallback: Redis → Fallback: In-memory
- Read-only mode on storage failure
- Graceful degradation

#### 3.4.3 Dead Letter Queue (Should Have)

**Failed Message Handling:**
- Store failed messages for later processing
- Configurable retention period
- Manual/automatic retry from DLQ
- Dead letter analysis and reporting

**DLQ Storage:**
- In-memory (limited size)
- Redis (distributed)
- PostgreSQL (persistent)

#### 3.4.4 Error Categories (Must Have)

**Transient Errors (Retryable):**
- Network timeouts
- Rate limit errors (429)
- Server errors (500, 502, 503, 504)
- Temporary unavailability

**Permanent Errors (Non-Retryable):**
- Invalid API key (401)
- Bad request (400)
- Not found (404)
- Validation errors

**Critical Errors (Alert + Fallback):**
- LLM provider completely down
- Storage corruption
- Security violations

#### 3.4.5 Circuit Breaker Enhancements (Must Have)

**Already Implemented:**
- Open/Closed/Half-Open states
- Failure threshold
- Reset timeout

**Enhancements Needed:**
- **Per-Dependency Circuit Breakers**: Separate for each LLM provider, storage backend
- **Health-Based State**: Auto-recovery based on health checks
- **Metrics Integration**: Circuit breaker state metrics
- **Gradual Recovery**: Slowly increase traffic in half-open state

### 3.5 Technical Design

#### 3.5.1 Architecture

```
Request → Error Classifier → Recovery Strategy
                                    ↓
                    ┌───────────────┴───────────────┐
                    ↓               ↓               ↓
                 Retry          Fallback          DLQ
```

#### 3.5.2 Package Structure

```
sage-adk/
├── core/resilience/          # (Already exists)
│   ├── retry.go              # Enhance with jitter, budget
│   ├── circuit_breaker.go    # Enhance with per-dependency
│   ├── fallback.go           # NEW: Fallback strategies
│   └── dlq.go                # NEW: Dead letter queue
├── core/recovery/            # NEW: Advanced recovery
│   ├── doc.go
│   ├── classifier.go         # Error classification
│   ├── strategy.go           # Recovery strategy selector
│   ├── llm_fallback.go       # LLM-specific fallbacks
│   └── storage_fallback.go   # Storage fallbacks
```

#### 3.5.3 Configuration

```go
type RecoveryConfig struct {
    // Retry enhancements
    Retry RetryConfig `yaml:"retry" json:"retry"`

    // Fallback strategies
    Fallback FallbackConfig `yaml:"fallback" json:"fallback"`

    // Dead letter queue
    DLQ DLQConfig `yaml:"dlq" json:"dlq"`

    // Circuit breaker enhancements
    CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" json:"circuit_breaker"`
}

type RetryConfig struct {
    // Existing fields...

    Jitter           bool    `yaml:"jitter" json:"jitter"`
    BackoffMultiplier float64 `yaml:"backoff_multiplier" json:"backoff_multiplier"`
    RetryBudget      int     `yaml:"retry_budget" json:"retry_budget"`
}

type FallbackConfig struct {
    Enabled bool `yaml:"enabled" json:"enabled"`

    // LLM fallback chain
    LLMChain []LLMFallback `yaml:"llm_chain" json:"llm_chain"`

    // Response fallbacks
    CachedResponses bool   `yaml:"cached_responses" json:"cached_responses"`
    DefaultMessage  string `yaml:"default_message" json:"default_message"`
}

type LLMFallback struct {
    Provider string  `yaml:"provider" json:"provider"` // openai, anthropic, gemini
    Model    string  `yaml:"model" json:"model"`
    Priority int     `yaml:"priority" json:"priority"` // Lower = higher priority
}

type DLQConfig struct {
    Enabled         bool   `yaml:"enabled" json:"enabled"`
    Storage         string `yaml:"storage" json:"storage"` // memory, redis, postgres
    MaxSize         int    `yaml:"max_size" json:"max_size"`
    RetentionHours  int    `yaml:"retention_hours" json:"retention_hours"`
    AutoRetry       bool   `yaml:"auto_retry" json:"auto_retry"`
    RetryInterval   int    `yaml:"retry_interval" json:"retry_interval"` // seconds
}
```

### 3.6 Success Metrics

- **Recovery Rate**: >95% of transient errors auto-recovered
- **Fallback Success**: >90% fallback success rate
- **DLQ Efficiency**: <5% messages end in DLQ
- **User Impact**: <1% user-facing errors

---

## 4. Performance Tuning

### 4.1 Overview

Optimize agent performance for production workloads with connection pooling, caching, and resource optimization.

### 4.2 Business Goals

- **Throughput**: Handle 10,000+ requests/second
- **Latency**: p95 < 100ms (excluding LLM)
- **Cost Efficiency**: Minimize resource usage
- **Scalability**: Linear scaling with resources

### 4.3 Requirements

#### 4.3.1 Connection Pooling (Must Have)

**HTTP Clients:**
- Configurable pool size
- Connection reuse
- Keep-alive tuning
- DNS caching

**Storage Connections:**
- PostgreSQL: Connection pooling (already implemented)
- Redis: Connection pooling (already implemented)
- Tuning: max open, max idle, max lifetime

#### 4.3.2 Caching (Must Have)

**Response Caching:**
- Cache identical LLM requests
- TTL-based expiration
- Cache key: hash(messages + model + temperature)
- Configurable cache size

**DID Resolution Caching:**
- Cache DID documents
- Blockchain query reduction
- TTL: 5-60 minutes

**Agent Card Caching:**
- Cache agent metadata
- Reduce protocol overhead

#### 4.3.3 Memory Optimization (Should Have)

**Object Pooling:**
- Message object reuse
- Byte buffer pooling
- Reduce GC pressure

**Streaming:**
- Stream large responses
- Chunk processing
- Backpressure handling

#### 4.3.4 Concurrency Tuning (Must Have)

**Goroutine Pooling:**
- Worker pool for message handling
- Bounded concurrency
- Configurable pool size

**Context Propagation:**
- Request-scoped contexts
- Timeout propagation
- Cancellation propagation

#### 4.3.5 Profiling & Benchmarking (Should Have)

**Built-in Profiling:**
- CPU profiling endpoint
- Memory profiling endpoint
- Goroutine profiling
- Block profiling

**Benchmarks:**
- Message handling benchmarks
- LLM adapter benchmarks
- Storage benchmarks
- Protocol benchmarks

### 4.4 Technical Design

#### 4.4.1 Package Structure

```
sage-adk/
├── performance/
│   ├── doc.go
│   ├── pool/
│   │   ├── worker.go         # Worker pool
│   │   ├── buffer.go         # Buffer pool
│   │   └── object.go         # Object pool
│   ├── cache/
│   │   ├── response.go       # Response cache
│   │   ├── did.go            # DID cache
│   │   └── agentcard.go      # Agent card cache
│   └── profiling/
│       ├── cpu.go            # CPU profiling
│       ├── memory.go         # Memory profiling
│       └── handler.go        # HTTP handlers
```

#### 4.4.2 Configuration

```go
type PerformanceConfig struct {
    // Connection pooling
    ConnectionPool ConnectionPoolConfig `yaml:"connection_pool" json:"connection_pool"`

    // Caching
    Cache CacheConfig `yaml:"cache" json:"cache"`

    // Concurrency
    Concurrency ConcurrencyConfig `yaml:"concurrency" json:"concurrency"`

    // Profiling
    Profiling ProfilingConfig `yaml:"profiling" json:"profiling"`
}

type ConnectionPoolConfig struct {
    MaxOpenConns    int `yaml:"max_open_conns" json:"max_open_conns"`
    MaxIdleConns    int `yaml:"max_idle_conns" json:"max_idle_conns"`
    ConnMaxLifetime int `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // seconds
    ConnMaxIdleTime int `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
}

type CacheConfig struct {
    Enabled         bool   `yaml:"enabled" json:"enabled"`
    ResponseCache   bool   `yaml:"response_cache" json:"response_cache"`
    DIDCache        bool   `yaml:"did_cache" json:"did_cache"`
    AgentCardCache  bool   `yaml:"agent_card_cache" json:"agent_card_cache"`
    MaxSize         int    `yaml:"max_size" json:"max_size"` // entries
    TTL             int    `yaml:"ttl" json:"ttl"`           // seconds
}

type ConcurrencyConfig struct {
    WorkerPoolSize  int `yaml:"worker_pool_size" json:"worker_pool_size"`
    MaxConcurrent   int `yaml:"max_concurrent" json:"max_concurrent"`
    QueueSize       int `yaml:"queue_size" json:"queue_size"`
}

type ProfilingConfig struct {
    Enabled  bool `yaml:"enabled" json:"enabled"`
    Port     int  `yaml:"port" json:"port"`
    CPUPath  string `yaml:"cpu_path" json:"cpu_path"`
    MemPath  string `yaml:"mem_path" json:"mem_path"`
}
```

### 4.5 Success Metrics

- **Throughput**: 10,000 RPS sustained
- **Latency**: p95 < 100ms (non-LLM), p99 < 200ms
- **Memory**: <500MB for 10,000 concurrent users
- **CPU**: <50% at 5,000 RPS

---

## 5. Production Deployment Guide

### 5.1 Overview

Comprehensive documentation for deploying SAGE ADK agents in production environments.

### 5.2 Requirements

#### 5.2.1 Deployment Architectures (Must Have)

**Single Instance:**
- Simple deployment
- No HA
- Good for: dev, staging, small prod

**Multi-Instance (Load Balanced):**
- Horizontal scaling
- HA with load balancer
- Stateless agents
- Good for: production

**Distributed (Microservices):**
- Agent per service
- Service mesh integration
- Good for: large-scale systems

**Serverless:**
- AWS Lambda, Google Cloud Functions
- Event-driven
- Good for: sporadic workloads

#### 5.2.2 Infrastructure (Must Have)

**Kubernetes Deployment:**
- Deployment manifests
- Service definitions
- ConfigMaps and Secrets
- Horizontal Pod Autoscaler
- Ingress configuration

**Docker:**
- Multi-stage Dockerfile
- Image optimization
- Security scanning
- Registry setup

**Cloud Providers:**
- AWS (ECS, EKS, Lambda)
- GCP (GKE, Cloud Run, Functions)
- Azure (AKS, Container Instances)

#### 5.2.3 Configuration Management (Must Have)

**Environment Variables:**
- Naming conventions
- Secret management
- Config validation

**Config Files:**
- YAML structure
- Environment-specific configs
- Config reloading

**Secret Management:**
- HashiCorp Vault
- AWS Secrets Manager
- GCP Secret Manager
- Kubernetes Secrets

#### 5.2.4 Monitoring Setup (Must Have)

**Prometheus:**
- Installation
- Scrape configuration
- Alert rules
- Grafana dashboards

**Logging:**
- ELK Stack setup
- Log aggregation
- Log retention
- Query examples

**Tracing:**
- Jaeger installation
- Trace collection
- Query and analysis

#### 5.2.5 Security (Must Have)

**TLS/SSL:**
- Certificate management
- HTTPS enforcement
- mTLS for agent-to-agent

**Authentication:**
- API key management
- JWT tokens
- OAuth2 integration

**Network Security:**
- Firewall rules
- Network policies
- VPC configuration

#### 5.2.6 Backup & Disaster Recovery (Should Have)

**Data Backup:**
- PostgreSQL backup
- Redis persistence
- Key backup

**Disaster Recovery:**
- RPO/RTO targets
- Failover procedures
- Recovery testing

### 5.3 Document Structure

```
docs/deployment/
├── README.md                       # Overview
├── architecture/
│   ├── single-instance.md
│   ├── multi-instance.md
│   ├── distributed.md
│   └── serverless.md
├── platforms/
│   ├── kubernetes/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── configmap.yaml
│   │   ├── hpa.yaml
│   │   └── ingress.yaml
│   ├── docker/
│   │   ├── Dockerfile
│   │   ├── docker-compose.yml
│   │   └── .dockerignore
│   ├── aws/
│   │   ├── ecs.md
│   │   ├── eks.md
│   │   └── lambda.md
│   ├── gcp/
│   │   ├── gke.md
│   │   ├── cloud-run.md
│   │   └── functions.md
│   └── azure/
│       └── aks.md
├── configuration/
│   ├── environment-variables.md
│   ├── config-files.md
│   └── secrets-management.md
├── monitoring/
│   ├── prometheus-setup.md
│   ├── grafana-dashboards/
│   ├── logging-setup.md
│   └── tracing-setup.md
├── security/
│   ├── tls-ssl.md
│   ├── authentication.md
│   └── network-security.md
├── backup-recovery/
│   ├── backup-strategy.md
│   └── disaster-recovery.md
└── troubleshooting/
    ├── common-issues.md
    └── debugging-guide.md
```

### 5.4 Success Metrics

- **Completeness**: Cover 100% of deployment scenarios
- **Clarity**: <30 min to deploy first agent
- **Reliability**: 99.9% deployment success rate
- **Community**: 10+ community deployment guides

---

## Priority & Timeline

### Phase 3A (4-6 weeks) - Critical Production Features

**Priority: P0 (Must Have for v0.2.0)**

1. **Monitoring & Observability** (2 weeks)
   - Week 1: Metrics (Prometheus) + Logging
   - Week 2: Health checks + Basic tracing

2. **Rate Limiting** (2 weeks)
   - Week 1: Core limiter + Token bucket
   - Week 2: Middleware + Storage backends

3. **Advanced Error Recovery** (1-2 weeks)
   - Week 1: Enhanced retry + LLM fallback
   - Week 2: DLQ + Circuit breaker enhancements

### Phase 3B (2-4 weeks) - Performance & Operations

**Priority: P1 (Nice to Have for v0.2.0)**

4. **Performance Tuning** (2 weeks)
   - Week 1: Caching + Connection pooling
   - Week 2: Profiling + Optimization

5. **Production Deployment Guide** (1-2 weeks)
   - Week 1: Core documentation + Kubernetes
   - Week 2: Cloud providers + Best practices

### Total Timeline: 6-10 weeks

**Milestones:**
- Week 2: Basic monitoring operational
- Week 4: Rate limiting + Recovery complete
- Week 6: Performance optimization done
- Week 8: Documentation complete
- Week 10: Production-ready release (v0.2.0)

---

## Success Criteria

**Technical:**
- ✅ 99.9% uptime in production
- ✅ p95 latency < 100ms (non-LLM)
- ✅ Handle 10,000 RPS
- ✅ <1% error rate
- ✅ 95%+ error auto-recovery

**Operational:**
- ✅ <5 min to deploy
- ✅ <1 hr to debug issues
- ✅ Zero cost overruns
- ✅ 24/7 monitoring

**Adoption:**
- ✅ All examples include observability
- ✅ 10+ production deployments
- ✅ Community deployment guides
- ✅ Zero security incidents

---

## Open Questions

1. **Tracing**: OpenTelemetry vs. custom solution?
2. **Logging**: Structured logger library (zap, zerolog, slog)?
3. **Cache**: In-memory (ristretto, freecache) vs. Redis only?
4. **Profiling**: Always-on with sampling or on-demand?
5. **Deployment**: Support Nomad, Docker Swarm?

---

## Appendix

### A. Related Documents

- [Development Roadmap](./DEVELOPMENT_ROADMAP_v1.0_20251006-235205.md)
- [Task Priority Matrix](./TASK_PRIORITY_MATRIX_v1.0_20251006-235205.md)
- [Architecture Overview](./architecture/overview.md)

### B. References

- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Go Performance Tuning](https://go.dev/doc/diagnostics)
- [Kubernetes Production Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
