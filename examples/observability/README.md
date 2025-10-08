# Observability Example

This example demonstrates how to use the SAGE ADK observability features to monitor, log, and health-check your agent.

## Features Demonstrated

- **Prometheus Metrics**: Collect and expose metrics for agent operations and LLM calls
- **Structured Logging**: JSON structured logging with context propagation
- **Health Checks**: Kubernetes-compatible liveness, readiness, and startup probes
- **HTTP Middleware**: Automatic request/response logging and metrics collection

## Running the Example

```bash
cd examples/observability
go run main.go
```

The example starts two HTTP servers:

- **Application Server** (`:8080`): Main application endpoints with observability middleware
- **Observability Server** (`:9090`): Metrics and health check endpoints

## Available Endpoints

### Application Endpoints (Port 8080)

| Endpoint | Description |
|----------|-------------|
| `GET /` | Main endpoint - demonstrates basic request handling |
| `GET /api/process` | API endpoint - demonstrates API request processing |

### Observability Endpoints (Port 9090)

| Endpoint | Description | Kubernetes Probe |
|----------|-------------|-----------------|
| `GET /metrics` | Prometheus metrics in OpenMetrics format | - |
| `GET /health/live` | Liveness probe - checks if agent is running | `livenessProbe` |
| `GET /health/ready` | Readiness probe - checks if agent can serve traffic | `readinessProbe` |
| `GET /health/startup` | Startup probe - checks if agent has completed initialization | `startupProbe` |

## Testing the Endpoints

### Test Application Endpoints

```bash
# Main endpoint
curl http://localhost:8080/

# API endpoint
curl http://localhost:8080/api/process
```

### Test Observability Endpoints

```bash
# View metrics
curl http://localhost:9090/metrics

# Check liveness (should always be healthy while running)
curl http://localhost:9090/health/live

# Check readiness (unhealthy until database connects ~3 seconds)
curl http://localhost:9090/health/ready

# Check startup (unhealthy for ~1 second, then healthy)
curl http://localhost:9090/health/startup
```

## Metrics Available

The example exposes the following metrics:

### Agent Metrics

- `sage_agent_requests_total` - Total number of requests processed
- `sage_agent_request_duration_seconds` - Request duration histogram
- `sage_agent_errors_total` - Total number of errors by type
- `sage_agent_messages_received_total` - Messages received by sender type
- `sage_agent_messages_sent_total` - Messages sent by recipient type

### LLM Metrics

- `sage_llm_api_calls_total` - Total LLM API calls by provider and model
- `sage_llm_api_latency_seconds` - LLM API call latency histogram
- `sage_llm_tokens_total` - Total tokens processed
- `sage_llm_tokens_prompt_total` - Prompt tokens processed
- `sage_llm_tokens_output_total` - Output/completion tokens processed

## Log Output

The example uses structured JSON logging. Each log entry includes:

```json
{
  "timestamp": "2025-10-08T12:34:56.789Z",
  "level": "info",
  "message": "Processing request",
  "agent_id": "example-agent",
  "path": "/api/process",
  "method": "GET"
}
```

## Health Check Behavior

The example demonstrates health check state transitions:

1. **Startup Phase** (0-1 second):
   - `startup`: unhealthy
   - `ready`: unhealthy
   - `live`: healthy

2. **Database Connection Phase** (1-3 seconds):
   - `startup`: healthy
   - `ready`: unhealthy (waiting for database)
   - `live`: healthy

3. **Ready to Serve** (3+ seconds):
   - `startup`: healthy
   - `ready`: healthy
   - `live`: healthy

## Kubernetes Integration

You can use these health check endpoints in Kubernetes:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-agent
spec:
  containers:
  - name: agent
    image: example-agent:latest
    ports:
    - containerPort: 8080
      name: http
    - containerPort: 9090
      name: metrics
    livenessProbe:
      httpGet:
        path: /health/live
        port: 9090
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 9090
      initialDelaySeconds: 5
      periodSeconds: 5
    startupProbe:
      httpGet:
        path: /health/startup
        port: 9090
      failureThreshold: 30
      periodSeconds: 10
```

## Prometheus Configuration

To scrape metrics from this example:

```yaml
scrape_configs:
  - job_name: 'example-agent'
    static_configs:
      - targets: ['localhost:9090']
```

## Code Walkthrough

### 1. Create Observability Manager

```go
manager, err := observability.NewManager(&observability.ManagerConfig{
    AgentID: "example-agent",
    Config:  config,
})
```

The manager provides:
- Logger for structured logging
- Metrics collectors (agent and LLM)
- Health checkers (liveness, readiness, startup)
- HTTP middleware

### 2. Add Custom Health Checks

```go
dbChecker := &DatabaseChecker{healthy: false}
manager.AddReadinessCheck(dbChecker)
```

Custom health checks can be added to the readiness probe to verify dependencies.

### 3. Use Middleware

```go
appHandler := manager.Middleware().Handler(appMux)
```

The middleware automatically:
- Logs all requests and responses
- Records request metrics (duration, count, errors)
- Propagates request context (request ID, agent ID)

### 4. Record Metrics

```go
// Record agent metrics
manager.AgentMetrics().RecordRequest("example-agent", "http", 0.1)

// Record LLM metrics
manager.LLMMetrics().RecordCall("openai", "gpt-4", 0.5)
manager.LLMMetrics().RecordTokens("openai", "gpt-4", 100, 200)
```

### 5. Structured Logging

```go
logger := manager.Logger()
logger.Info(ctx, "Processing request",
    logging.String("path", r.URL.Path),
    logging.String("method", r.Method))
```

## Next Steps

- Integrate with your agent implementation
- Add custom health checks for your dependencies
- Configure Prometheus to scrape metrics
- Set up Grafana dashboards for visualization
- Configure log aggregation (e.g., ELK stack)

## See Also

- [Observability Package Documentation](../../observability/doc.go)
- [Metrics Documentation](../../observability/metrics/doc.go)
- [Logging Documentation](../../observability/logging/doc.go)
- [Health Checks Documentation](../../observability/health/doc.go)
