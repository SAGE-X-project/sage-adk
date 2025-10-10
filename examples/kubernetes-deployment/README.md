## Kubernetes Deployment Example

Production-ready Kubernetes deployment configuration for SAGE ADK agents with high availability, auto-scaling, and observability.

##  Files

- `namespace.yaml` - Namespace definition
- `serviceaccount.yaml` - RBAC configuration
- `configmap.yaml` - Agent configuration
- `secret.yaml` - Sensitive data (API keys, passwords)
- `deployment.yaml` - Main agent deployment
- `service.yaml` - Service definitions
- `ingress.yaml` - Ingress configuration
- `hpa.yaml` - Horizontal Pod Autoscaler
- `Dockerfile` - Container image definition

##  Quick Start

### 1. Build Docker Image

```bash
# Build the image
docker build -t sage-adk-agent:v1.1.0 .

# Tag for your registry
docker tag sage-adk-agent:v1.1.0 your-registry/sage-adk-agent:v1.1.0

# Push to registry
docker push your-registry/sage-adk-agent:v1.1.0
```

### 2. Create Secrets

```bash
# Create namespace
kubectl apply -f namespace.yaml

# Create secrets with your API keys
kubectl create secret generic sage-adk-secrets \
  --namespace=sage-adk \
  --from-literal=openai-api-key=YOUR_OPENAI_KEY \
  --from-literal=anthropic-api-key=YOUR_ANTHROPIC_KEY \
  --from-literal=gemini-api-key=YOUR_GEMINI_KEY
```

### 3. Deploy to Kubernetes

```bash
# Apply all configurations
kubectl apply -f namespace.yaml
kubectl apply -f serviceaccount.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f ingress.yaml

# Or apply all at once
kubectl apply -f .
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n sage-adk

# Check services
kubectl get svc -n sage-adk

# Check ingress
kubectl get ingress -n sage-adk

# View logs
kubectl logs -n sage-adk -l app=sage-adk-agent -f

# Port forward for local testing
kubectl port-forward -n sage-adk svc/sage-adk-agent 8080:80
```

##  Configuration

### Environment Variables

Key configurations in `configmap.yaml`:

```yaml
# Protocol Selection
protocol.mode: "auto"  # auto, a2a, or sage

# Storage Backend
storage.type: "redis"  # memory, redis, or postgres

# LLM Provider
llm.provider: "openai"  # openai, anthropic, or gemini
llm.model: "gpt-4"
```

### Resource Limits

Configured in `deployment.yaml`:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Auto-Scaling

HPA configuration in `hpa.yaml`:

- Min replicas: 3
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

##  Architecture

```

   Ingress   

       

  Load Balancer      
  (Service)          

       

  Agent Pods (3-10)  
  - Liveness Probe   
  - Readiness Probe  
  - Startup Probe    

       

  Storage Backend    
  (Redis/PostgreSQL) 

```

##  Security Features

### 1. Pod Security

```yaml
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  capabilities:
    drop:
    - ALL
```

### 2. Network Policies

Create `networkpolicy.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: sage-adk-agent
  namespace: sage-adk
spec:
  podSelector:
    matchLabels:
      app: sage-adk-agent
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS for LLM APIs
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

### 3. Secret Management

```bash
# Use external secrets operator (recommended)
kubectl apply -f https://raw.githubusercontent.com/external-secrets/external-secrets/main/deploy/crds/bundle.yaml

# Or use sealed secrets
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml
```

##  Monitoring

### Prometheus Integration

The deployment exposes Prometheus metrics at `/metrics`:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

Create `servicemonitor.yaml`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: sage-adk-agent
  namespace: sage-adk
spec:
  selector:
    matchLabels:
      app: sage-adk-agent
  endpoints:
  - port: metrics
    interval: 30s
```

### Grafana Dashboards

Import dashboard from `../monitoring-setup/grafana-dashboard.json`

##  Health Checks

Three types of probes configured:

### 1. Liveness Probe
- Checks if pod is alive
- Endpoint: `/health/live`
- Failure: Pod restart

### 2. Readiness Probe
- Checks if pod can serve traffic
- Endpoint: `/health/ready`
- Failure: Removed from load balancer

### 3. Startup Probe
- Checks if app has started
- Endpoint: `/health/startup`
- Failure: Pod restart after 30 attempts

##  Rolling Updates

```bash
# Update image
kubectl set image deployment/sage-adk-agent \
  sage-adk-agent=your-registry/sage-adk-agent:v1.2.0 \
  -n sage-adk

# Check rollout status
kubectl rollout status deployment/sage-adk-agent -n sage-adk

# Rollback if needed
kubectl rollout undo deployment/sage-adk-agent -n sage-adk
```

##  Scaling

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl scale deployment sage-adk-agent --replicas=5 -n sage-adk
```

### Auto-Scaling

HPA automatically scales based on:
- CPU utilization (70%)
- Memory utilization (80%)
- Custom metrics (requests/sec)

##  Troubleshooting

### Pod not starting

```bash
# Describe pod
kubectl describe pod -n sage-adk -l app=sage-adk-agent

# Check logs
kubectl logs -n sage-adk -l app=sage-adk-agent --tail=100

# Check events
kubectl get events -n sage-adk --sort-by='.lastTimestamp'
```

### Connectivity issues

```bash
# Test service
kubectl run -it --rm debug --image=alpine --restart=Never -n sage-adk -- sh
apk add curl
curl http://sage-adk-agent/health

# Check DNS
nslookup sage-adk-agent.sage-adk.svc.cluster.local
```

### Performance issues

```bash
# Check resource usage
kubectl top pods -n sage-adk

# Check HPA status
kubectl get hpa -n sage-adk

# View metrics
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/sage-adk/pods
```

##  Multi-Region Deployment

For multi-region setup:

1. Deploy to multiple clusters
2. Use global load balancer (e.g., GCP Global LB, AWS Route53)
3. Configure cross-region replication for storage
4. Use external DNS for automatic DNS management

##  Dependencies

### Required

- Kubernetes 1.24+
- Metrics Server (for HPA)
- Ingress Controller (NGINX recommended)

### Optional

- Prometheus Operator (for monitoring)
- Cert-Manager (for TLS certificates)
- External Secrets Operator (for secret management)

##  Production Checklist

- [ ] Secrets properly configured
- [ ] Resource limits set
- [ ] Health checks configured
- [ ] Monitoring enabled
- [ ] Logging configured
- [ ] Backup strategy for storage
- [ ] Disaster recovery plan
- [ ] Security policies applied
- [ ] Network policies configured
- [ ] TLS certificates installed
- [ ] Auto-scaling tested
- [ ] Rolling update strategy tested

##  Additional Resources

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Production Deployment Guide](../../docs/deployment/)
- [Monitoring Setup](../monitoring-setup/)
- [SAGE ADK Documentation](../../README.md)

## License

LGPL-3.0-or-later
