# Production Deployment Guide

**Version:** 1.0
**Last Updated:** 2025-10-08
**Status:** Draft

## Overview

This guide provides comprehensive instructions for deploying SAGE ADK agents in production environments. It covers deployment architectures, platform-specific configurations, monitoring setup, security best practices, and operational procedures.

## Table of Contents

### 1. [Architecture Patterns](./architecture/)
- [Single Instance Deployment](./architecture/single-instance.md)
- [Multi-Instance with Load Balancing](./architecture/multi-instance.md)
- [Distributed Microservices](./architecture/distributed.md)
- [Serverless Deployment](./architecture/serverless.md)

### 2. [Platform Guides](./platforms/)
- [Kubernetes](./platforms/kubernetes/)
- [Docker](./platforms/docker/)
- [AWS](./platforms/aws/)
- [Google Cloud](./platforms/gcp/)
- [Azure](./platforms/azure/)

### 3. [Configuration Management](./configuration/)
- [Environment Variables](./configuration/environment-variables.md)
- [Config Files](./configuration/config-files.md)
- [Secrets Management](./configuration/secrets-management.md)

### 4. [Monitoring & Observability](./monitoring/)
- [Prometheus Setup](./monitoring/prometheus-setup.md)
- [Grafana Dashboards](./monitoring/grafana-dashboards/)
- [Logging Setup](./monitoring/logging-setup.md)
- [Distributed Tracing](./monitoring/tracing-setup.md)

### 5. [Security](./security/)
- [TLS/SSL Configuration](./security/tls-ssl.md)
- [Authentication](./security/authentication.md)
- [Network Security](./security/network-security.md)

### 6. [Backup & Disaster Recovery](./backup-recovery/)
- [Backup Strategy](./backup-recovery/backup-strategy.md)
- [Disaster Recovery Plan](./backup-recovery/disaster-recovery.md)

### 7. [Troubleshooting](./troubleshooting/)
- [Common Issues](./troubleshooting/common-issues.md)
- [Debugging Guide](./troubleshooting/debugging-guide.md)

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker (for containerized deployments)
- Kubernetes cluster (for K8s deployments)
- Cloud provider account (for cloud deployments)

### Basic Deployment Steps

1. **Build the Agent**
   ```bash
   go build -o sage-agent ./cmd/agent
   ```

2. **Configure Environment**
   ```bash
   export OPENAI_API_KEY="your-api-key"
   export SAGE_OBSERVABILITY_ENABLED=true
   export SAGE_METRICS_PORT=9090
   ```

3. **Deploy**
   - [Docker](#docker-deployment)
   - [Kubernetes](#kubernetes-deployment)
   - [Cloud Provider](#cloud-deployment)

4. **Verify**
   ```bash
   curl http://localhost:8080/health/ready
   curl http://localhost:9090/metrics
   ```

## Deployment Checklist

### Pre-Deployment

- [ ] Environment variables configured
- [ ] Secrets stored securely (Vault, AWS Secrets Manager, etc.)
- [ ] TLS certificates obtained
- [ ] Database/Storage backends provisioned
- [ ] Monitoring stack set up
- [ ] Load balancer configured (if multi-instance)
- [ ] DNS records created
- [ ] Firewall rules configured

### Deployment

- [ ] Build and tag container image
- [ ] Push image to registry
- [ ] Deploy to target environment
- [ ] Verify health checks pass
- [ ] Test basic functionality
- [ ] Monitor logs for errors

### Post-Deployment

- [ ] Configure alerting rules
- [ ] Set up automated backups
- [ ] Document runbook procedures
- [ ] Train operations team
- [ ] Implement monitoring dashboards
- [ ] Schedule regular health checks

## Architecture Decision Guide

### When to Use Single Instance

**Best For:**
- Development/staging environments
- Low-traffic applications (<100 RPS)
- Proof of concepts
- Cost-sensitive deployments

**Pros:**
- Simple setup
- Low operational overhead
- Easy debugging

**Cons:**
- No high availability
- Limited scalability
- Single point of failure

### When to Use Multi-Instance

**Best For:**
- Production environments
- Medium to high traffic (100-10,000 RPS)
- High availability requirements
- Geographic distribution

**Pros:**
- High availability
- Horizontal scalability
- Load distribution
- Rolling updates

**Cons:**
- More complex setup
- Stateless design required
- Higher costs

### When to Use Distributed Microservices

**Best For:**
- Large-scale systems
- Multiple agent types
- Service mesh environments
- Complex workflows

**Pros:**
- Fine-grained scaling
- Service isolation
- Technology flexibility
- Team autonomy

**Cons:**
- High complexity
- Network overhead
- Distributed debugging
- Operational burden

### When to Use Serverless

**Best For:**
- Sporadic workloads
- Event-driven architectures
- Cost optimization
- Auto-scaling requirements

**Pros:**
- Zero management
- Pay per use
- Auto-scaling
- High availability

**Cons:**
- Cold start latency
- Execution time limits
- Vendor lock-in
- Limited control

## Platform Comparison

| Feature | Kubernetes | Docker | AWS Lambda | GCP Cloud Run | Azure Container Instances |
|---------|-----------|--------|------------|---------------|---------------------------|
| **Scaling** | Horizontal Pod Autoscaler | Manual | Auto | Auto | Manual/Auto |
| **HA** | Built-in | Swarm/Compose | Built-in | Built-in | Availability Zones |
| **Cost** | Medium | Low | Pay-per-use | Pay-per-use | Pay-per-use |
| **Complexity** | High | Low | Medium | Low | Low |
| **Portability** | High | High | Low | Medium | Low |
| **Best For** | Production | Dev/Test | Event-driven | Web services | Batch jobs |

## Configuration Approaches

### 1. Environment Variables (Recommended for Production)

**Pros:**
- Cloud-native
- Easy to change per environment
- Secure (no files to leak)
- 12-factor app compliant

**Cons:**
- Can become unwieldy with many vars
- Limited type support (all strings)

**Example:**
```bash
export SAGE_AGENT_ID="agent-001"
export SAGE_LLM_PROVIDER="openai"
export SAGE_LLM_API_KEY="${SECRET_OPENAI_KEY}"
export SAGE_STORAGE_TYPE="postgres"
export SAGE_DB_CONNECTION_STRING="${SECRET_DB_URL}"
```

### 2. YAML Configuration Files

**Pros:**
- Structured and readable
- Version control friendly
- Complex config support
- Environment-specific files

**Cons:**
- Must secure sensitive data
- Deployment overhead
- File management

**Example:**
```yaml
# config.prod.yaml
agent:
  id: agent-001
  protocol: sage

llm:
  provider: openai
  model: gpt-4

storage:
  type: postgres
  host: db.example.com
  port: 5432

observability:
  metrics:
    enabled: true
    port: 9090
  logging:
    level: info
    format: json
```

### 3. Secrets Management (Required for Production)

**Tools:**
- HashiCorp Vault
- AWS Secrets Manager
- GCP Secret Manager
- Azure Key Vault
- Kubernetes Secrets

**Best Practices:**
- Never commit secrets to git
- Rotate secrets regularly
- Use least privilege access
- Audit secret access
- Encrypt at rest and in transit

## Monitoring Setup

### Metrics (Prometheus)

**Installation:**
```bash
# Kubernetes
kubectl apply -f https://raw.githubusercontent.com/prometheus/prometheus/main/documentation/examples/prometheus-kubernetes.yml

# Docker
docker run -d -p 9090:9090 -v /path/to/prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus
```

**Scrape Config:**
```yaml
scrape_configs:
  - job_name: 'sage-agents'
    static_configs:
      - targets: ['agent-1:9090', 'agent-2:9090']
    metrics_path: /metrics
    scrape_interval: 15s
```

**Key Metrics:**
- `sage_agent_requests_total` - Total requests
- `sage_agent_request_duration_seconds` - Latency
- `sage_agent_errors_total` - Error count
- `sage_llm_api_calls_total` - LLM usage
- `sage_llm_tokens_total` - Token usage

### Logging (ELK/Loki)

**Log Format:**
```json
{
  "timestamp": "2025-10-08T10:30:00Z",
  "level": "info",
  "message": "message handled",
  "agent_id": "agent-001",
  "request_id": "req-123",
  "trace_id": "trace-456",
  "duration_ms": 42
}
```

**Aggregation:**
- Elasticsearch + Kibana (ELK)
- Grafana Loki (recommended for K8s)
- CloudWatch Logs (AWS)
- Cloud Logging (GCP)

### Tracing (Jaeger)

**Setup:**
```bash
# All-in-one deployment
docker run -d -p 16686:16686 -p 6831:6831/udp jaegertracing/all-in-one:latest
```

**Agent Config:**
```yaml
tracing:
  enabled: true
  endpoint: http://jaeger:14268/api/traces
  service_name: sage-agent
  sampling_rate: 0.1  # Sample 10% of traces
```

## Security Checklist

### Transport Security

- [ ] TLS 1.3 enabled
- [ ] Valid SSL certificates
- [ ] HTTPS enforced (redirect HTTP)
- [ ] mTLS for agent-to-agent (SAGE protocol)
- [ ] Certificate rotation automated

### Authentication & Authorization

- [ ] API keys for LLM providers secured
- [ ] Agent-to-agent authentication (DID for SAGE)
- [ ] Admin endpoints protected
- [ ] Rate limiting enabled
- [ ] RBAC configured (if applicable)

### Network Security

- [ ] Firewall rules configured
- [ ] VPC/Network isolation
- [ ] Security groups defined
- [ ] DDoS protection enabled
- [ ] WAF configured (if public)

### Data Security

- [ ] Database encryption at rest
- [ ] Message encryption in transit
- [ ] Secrets encrypted (Vault, KMS)
- [ ] PII handling compliant
- [ ] Audit logging enabled

### Compliance

- [ ] GDPR compliance (if EU users)
- [ ] Data retention policies
- [ ] Privacy policy documented
- [ ] Security incident response plan
- [ ] Regular security audits

## Operational Runbooks

### Deployment Procedure

1. **Pre-deployment checks**
   - Run tests: `make test`
   - Build image: `docker build -t sage-agent:v1.2.3 .`
   - Security scan: `docker scan sage-agent:v1.2.3`

2. **Staging deployment**
   - Deploy to staging
   - Run smoke tests
   - Monitor for 1 hour

3. **Production deployment**
   - Blue-green or canary deployment
   - Monitor metrics closely
   - Be ready to rollback

4. **Post-deployment**
   - Verify health checks
   - Check logs for errors
   - Update documentation

### Rollback Procedure

1. **Trigger conditions**
   - Error rate > 5%
   - Latency > 2x baseline
   - Health checks failing
   - Customer impact

2. **Rollback steps**
   ```bash
   # Kubernetes
   kubectl rollout undo deployment/sage-agent

   # Docker
   docker service update --rollback sage-agent
   ```

3. **Post-rollback**
   - Investigate root cause
   - Fix in development
   - Test thoroughly
   - Redeploy

### Incident Response

1. **Detection**
   - Alert received
   - User report
   - Monitoring anomaly

2. **Assessment**
   - Severity: P1 (critical), P2 (major), P3 (minor)
   - Impact: users affected, services down
   - Root cause hypothesis

3. **Mitigation**
   - Immediate fix (rollback, scale, etc.)
   - Temporary workaround
   - Customer communication

4. **Resolution**
   - Permanent fix deployed
   - Monitoring verified
   - Post-mortem scheduled

5. **Post-mortem**
   - Timeline documented
   - Root cause identified
   - Action items created
   - Lessons learned shared

## Performance Tuning

### Baseline Metrics

**Target Performance:**
- Throughput: 10,000 RPS
- Latency: p95 < 100ms (excluding LLM)
- Error rate: < 0.1%
- Availability: 99.9%

### Optimization Techniques

1. **Connection Pooling**
   ```yaml
   storage:
     postgres:
       max_open_conns: 25
       max_idle_conns: 5
       conn_max_lifetime: 5m
   ```

2. **Caching**
   ```yaml
   performance:
     cache:
       enabled: true
       response_cache: true
       did_cache: true
       ttl: 300  # 5 minutes
   ```

3. **Concurrency**
   ```yaml
   performance:
     concurrency:
       worker_pool_size: 100
       max_concurrent: 1000
       queue_size: 10000
   ```

4. **Resource Limits**
   ```yaml
   # Kubernetes
   resources:
     requests:
       memory: "256Mi"
       cpu: "250m"
     limits:
       memory: "512Mi"
       cpu: "500m"
   ```

### Profiling

**CPU Profiling:**
```bash
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

**Memory Profiling:**
```bash
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof mem.prof
```

**Goroutine Analysis:**
```bash
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

## Cost Optimization

### LLM Cost Management

- Use cheaper models for simple tasks
- Implement response caching
- Set token limits
- Monitor usage per user/tenant
- Use prompt optimization

### Infrastructure Costs

- Right-size instances
- Use spot/preemptible instances
- Implement auto-scaling
- Schedule non-prod environments
- Use reserved instances for stable workloads

### Monitoring Costs

- Limit metric cardinality
- Use sampling for traces
- Set log retention policies
- Aggregate before exporting

## Troubleshooting

### Common Issues

#### Issue: High Latency

**Symptoms:**
- p95 latency > 500ms
- Slow response times

**Diagnosis:**
- Check LLM API latency
- Review database query times
- Analyze trace data
- Profile CPU usage

**Resolution:**
- Enable response caching
- Optimize database queries
- Increase worker pool size
- Scale horizontally

#### Issue: Memory Leak

**Symptoms:**
- Memory usage growing over time
- OOMKilled errors

**Diagnosis:**
- Memory profiling
- Goroutine leak check
- Connection leak check

**Resolution:**
- Fix goroutine leaks
- Implement connection pooling
- Add memory limits
- Restart periodically (temporary)

#### Issue: Rate Limit Errors

**Symptoms:**
- 429 errors from LLM provider
- Requests failing

**Diagnosis:**
- Check LLM usage metrics
- Review rate limit config

**Resolution:**
- Implement request queuing
- Use multiple API keys
- Add fallback providers
- Reduce request rate

### Debug Mode

**Enable verbose logging:**
```bash
export SAGE_LOG_LEVEL=debug
export SAGE_LOG_OUTPUT=stdout
```

**Enable profiling:**
```bash
export SAGE_PROFILING_ENABLED=true
export SAGE_PROFILING_PORT=6060
```

**Enable request tracing:**
```bash
export SAGE_TRACING_ENABLED=true
export SAGE_TRACING_SAMPLING_RATE=1.0  # 100% for debugging
```

## Support Resources

### Documentation
- [Architecture Overview](../architecture/overview.md)
- [API Reference](../api/)
- [Examples](../../examples/)

### Community
- GitHub Issues: https://github.com/sage-x-project/agent-develope-kit/issues
- Discord: [Coming Soon]
- Stack Overflow: Tag `sage-adk`

### Professional Support
- Enterprise Support: support@sage-x-project.org
- Consulting: consulting@sage-x-project.org

## Appendix

### A. Environment Variables Reference

See [Environment Variables Guide](./configuration/environment-variables.md)

### B. Configuration Schema

See [Configuration Reference](./configuration/config-files.md)

### C. Metrics Reference

See [Metrics Guide](./monitoring/prometheus-setup.md)

### D. API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health/live` | GET | Liveness probe |
| `/health/ready` | GET | Readiness probe |
| `/health/startup` | GET | Startup probe |
| `/metrics` | GET | Prometheus metrics |
| `/debug/pprof/` | GET | Profiling data |

---

**Last Updated:** 2025-10-08
**Version:** 1.0
**Contributors:** SAGE ADK Team
