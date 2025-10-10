# Monitoring Setup Example

Complete monitoring stack for SAGE ADK agents with Prometheus, Grafana, and Alertmanager.

## 📊 Components

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Alertmanager**: Alert routing and notification
- **Node Exporter**: System metrics

## 🚀 Quick Start

### 1. Start the Monitoring Stack

```bash
# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f
```

### 2. Access the Services

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093
- **SAGE ADK Agent**: http://localhost:8080
- **Agent Metrics**: http://localhost:8080/metrics

### 3. View Dashboards

1. Open Grafana at http://localhost:3000
2. Login with admin/admin
3. Navigate to "SAGE ADK" folder
4. Open "SAGE ADK Agent Overview" dashboard

## 📈 Available Dashboards

### 1. Agent Overview Dashboard

Displays:
- Request rate (req/sec)
- Error rate
- Response time (P50, P95, P99)
- Memory usage
- CPU usage
- Active goroutines

### 2. LLM Performance Dashboard

Monitors:
- LLM API latency
- Token usage
- API errors
- Rate limiting events

### 3. Storage Dashboard

Tracks:
- Storage operations (Get, Set, Delete)
- Cache hit/miss ratio
- Connection pool usage
- Storage errors

### 4. System Dashboard

Shows:
- CPU usage
- Memory usage
- Disk I/O
- Network traffic

## 🔔 Alerts

### Configured Alerts

| Alert | Severity | Threshold | Action |
|-------|----------|-----------|--------|
| HighErrorRate | Warning | >5% errors | Email team |
| CriticalErrorRate | Critical | >10% errors | Email oncall + Slack |
| HighResponseTime | Warning | P95 > 1s | Email team |
| AgentDown | Critical | Down > 1min | Email oncall + PagerDuty |
| HighMemoryUsage | Warning | >90% | Email team |
| StorageConnectionErrors | Critical | Any errors | Email oncall |

### Testing Alerts

```bash
# Trigger high error rate alert
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/messages \
    -H "Content-Type: application/json" \
    -d '{"invalid": "message"}' || true
done

# Check alert status
curl http://localhost:9093/api/v2/alerts

# View in Alertmanager UI
open http://localhost:9093
```

## 📊 Metrics Reference

### Request Metrics

```promql
# Total requests
sage_adk_requests_total

# Request rate
rate(sage_adk_requests_total[5m])

# Error rate
rate(sage_adk_requests_total{status="error"}[5m]) /
rate(sage_adk_requests_total[5m])

# Response time histogram
sage_adk_request_duration_seconds_bucket

# P95 latency
histogram_quantile(0.95, rate(sage_adk_request_duration_seconds_bucket[5m]))
```

### LLM Metrics

```promql
# LLM API calls
sage_adk_llm_requests_total

# LLM latency
sage_adk_llm_duration_seconds

# Token usage
sage_adk_llm_tokens_total

# LLM errors
sage_adk_llm_errors_total
```

### Storage Metrics

```promql
# Storage operations
sage_adk_storage_operations_total

# Storage latency
sage_adk_storage_duration_seconds

# Storage errors
sage_adk_storage_errors_total
```

### System Metrics

```promql
# Memory usage
go_memstats_alloc_bytes

# CPU usage
rate(process_cpu_seconds_total[5m])

# Goroutines
go_goroutines

# Garbage collection
rate(go_gc_duration_seconds_count[5m])
```

## 🔍 Useful Queries

### Performance Analysis

```promql
# Requests per second by status
sum(rate(sage_adk_requests_total[5m])) by (status)

# Average response time
avg(rate(sage_adk_request_duration_seconds_sum[5m])) /
avg(rate(sage_adk_request_duration_seconds_count[5m]))

# Top 5 slowest endpoints
topk(5,
  histogram_quantile(0.95,
    rate(sage_adk_request_duration_seconds_bucket[5m])
  )
) by (endpoint)
```

### Error Analysis

```promql
# Error rate percentage
100 * (
  rate(sage_adk_requests_total{status="error"}[5m]) /
  rate(sage_adk_requests_total[5m])
)

# Errors by type
sum(rate(sage_adk_requests_total{status="error"}[5m])) by (error_type)
```

### Capacity Planning

```promql
# Memory growth rate
deriv(go_memstats_alloc_bytes[1h])

# Request trend
predict_linear(sage_adk_requests_total[1h], 3600)

# Saturation
(go_memstats_alloc_bytes / go_memstats_sys_bytes) * 100
```

## 📧 Alert Notifications

### Email Setup

Edit `alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@example.com'
  smtp_auth_username: 'alerts@example.com'
  smtp_auth_password: 'your-app-password'
```

### Slack Integration

Add to `alertmanager.yml`:

```yaml
receivers:
  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#critical-alerts'
        title: 'SAGE ADK Critical Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}'
```

### PagerDuty Integration

```yaml
receivers:
  - name: 'critical-alerts'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
        description: '{{ .GroupLabels.alertname }}'
```

## 🛠️ Advanced Configuration

### Custom Metrics

Add to your agent code:

```go
import "github.com/sage-x-project/sage-adk/observability/metrics"

// Counter
requestCounter := metrics.NewCounter(
    "custom_requests_total",
    "Total custom requests",
)
requestCounter.Inc()

// Gauge
activeUsers := metrics.NewGauge(
    "active_users",
    "Number of active users",
)
activeUsers.Set(42)

// Histogram
latency := metrics.NewHistogram(
    "custom_latency_seconds",
    "Custom operation latency",
)
latency.Observe(0.123)
```

### Recording Rules

Add to `prometheus.yml`:

```yaml
rule_files:
  - '/etc/prometheus/recording-rules.yml'
```

Create `recording-rules.yml`:

```yaml
groups:
  - name: sage_adk_recording
    interval: 30s
    rules:
      - record: job:sage_adk_requests:rate5m
        expr: rate(sage_adk_requests_total[5m])

      - record: job:sage_adk_error_rate:rate5m
        expr: |
          rate(sage_adk_requests_total{status="error"}[5m]) /
          rate(sage_adk_requests_total[5m])
```

### Long-term Storage

For production, consider:

- **Thanos**: Long-term Prometheus storage
- **Cortex**: Multi-tenant Prometheus
- **VictoriaMetrics**: High-performance alternative

## 🐛 Troubleshooting

### Metrics Not Appearing

```bash
# Check if agent is exposing metrics
curl http://localhost:8080/metrics

# Check Prometheus targets
open http://localhost:9090/targets

# Check Prometheus logs
docker-compose logs prometheus
```

### Alerts Not Firing

```bash
# Test alert rules
docker-compose exec prometheus promtool check rules /etc/prometheus/alert-rules.yml

# Check Alertmanager config
docker-compose exec alertmanager amtool check-config /etc/alertmanager/alertmanager.yml

# Force alert evaluation
curl -X POST http://localhost:9090/-/reload
```

### Grafana Dashboard Issues

```bash
# Check datasource
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:3000/api/datasources

# Reload provisioning
docker-compose restart grafana
```

## 📊 Grafana Tips

### Creating Custom Dashboards

1. Click "+" → "Dashboard"
2. Add Panel
3. Select metrics from Prometheus
4. Apply transformations
5. Save dashboard

### Useful Panel Types

- **Graph**: Time-series data
- **Stat**: Single value
- **Gauge**: Progress indicator
- **Table**: Tabular data
- **Heatmap**: Distribution over time

### Variables

Create dashboard variables for:
- `$instance`: Agent instance
- `$interval`: Time range
- `$environment`: Environment (prod/staging/dev)

## 🔐 Security

### Secure Grafana

```yaml
# In docker-compose.yaml
environment:
  - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
  - GF_USERS_ALLOW_SIGN_UP=false
  - GF_AUTH_ANONYMOUS_ENABLED=false
```

### Secure Prometheus

Add authentication:

```yaml
# web-config.yml
basic_auth_users:
  admin: $2y$10$hashed_password_here
```

## 📚 Learn More

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [SAGE ADK Metrics](../../observability/metrics/doc.go)

## License

LGPL-3.0-or-later
