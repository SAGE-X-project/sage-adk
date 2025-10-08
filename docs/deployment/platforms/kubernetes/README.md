# Kubernetes Deployment Guide

**Version:** 1.0
**Last Updated:** 2025-10-08

## Overview

This guide provides instructions for deploying SAGE ADK agents on Kubernetes. It covers deployment manifests, configuration, scaling, and best practices.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Container registry access
- Helm 3.x (optional)

## Quick Start

### 1. Create Namespace

```bash
kubectl create namespace sage-system
```

### 2. Create Secrets

```bash
# Create secrets from environment variables
kubectl create secret generic sage-agent-secrets \
  --from-literal=openai-api-key=$OPENAI_API_KEY \
  --from-literal=postgres-user=sage \
  --from-literal=postgres-password=$POSTGRES_PASSWORD \
  -n sage-system
```

### 3. Apply Manifests

```bash
# Apply in order
kubectl apply -f configmap.yaml
kubectl apply -f secrets.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f ingress.yaml
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n sage-system

# Check deployment
kubectl get deployment sage-agent -n sage-system

# Check service
kubectl get svc sage-agent -n sage-system

# Check logs
kubectl logs -f deployment/sage-agent -n sage-system
```

## Files Overview

| File | Purpose |
|------|---------|
| `deployment.yaml` | Main deployment manifest |
| `service.yaml` | Service definitions (LoadBalancer, ClusterIP) |
| `configmap.yaml` | Configuration data |
| `secrets.yaml` | Sensitive data (template) |
| `hpa.yaml` | Horizontal Pod Autoscaler |
| `ingress.yaml` | Ingress rules (Nginx, Istio, ALB, GCE) |
| `namespace.yaml` | Namespace definition |
| `rbac.yaml` | Service account, roles, bindings |
| `networkpolicy.yaml` | Network policies |
| `pdb.yaml` | Pod Disruption Budget |

## Configuration

### Environment Variables

Set in `deployment.yaml`:

```yaml
env:
- name: SAGE_LLM_PROVIDER
  value: "openai"
- name: OPENAI_API_KEY
  valueFrom:
    secretKeyRef:
      name: sage-agent-secrets
      key: openai-api-key
```

### ConfigMap

Defined in `configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sage-agent-config
data:
  llm-provider: "openai"
  log-level: "info"
```

### Secrets

**Option 1: kubectl create**
```bash
kubectl create secret generic sage-agent-secrets \
  --from-literal=openai-api-key=sk-... \
  -n sage-system
```

**Option 2: Sealed Secrets (recommended)**
```bash
# Install Sealed Secrets
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# Create sealed secret
echo -n 'sk-...' | kubectl create secret generic sage-agent-secrets \
  --dry-run=client --from-file=openai-api-key=/dev/stdin -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

kubectl apply -f sealed-secret.yaml
```

**Option 3: External Secrets Operator (production)**
```bash
# Install ESO
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace

# Configure AWS Secrets Manager
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: sage-system
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: sage-agent
EOF

# Create ExternalSecret
kubectl apply -f external-secret.yaml
```

## Scaling

### Horizontal Pod Autoscaler (HPA)

Automatically scales based on CPU, memory, or custom metrics:

```yaml
# hpa.yaml
spec:
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        averageUtilization: 70
```

**Monitor HPA:**
```bash
kubectl get hpa sage-agent-hpa -n sage-system --watch
```

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl scale deployment sage-agent --replicas=5 -n sage-system
```

### Vertical Pod Autoscaler (VPA)

Automatically adjusts resource requests/limits:

```bash
# Install VPA
git clone https://github.com/kubernetes/autoscaler.git
cd autoscaler/vertical-pod-autoscaler
./hack/vpa-up.sh

# Apply VPA
kubectl apply -f vpa.yaml
```

## Ingress Configuration

### Nginx Ingress

```bash
# Install Nginx Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml

# Apply Ingress
kubectl apply -f ingress.yaml

# Get external IP
kubectl get svc ingress-nginx-controller -n ingress-nginx
```

### Istio Gateway

```bash
# Install Istio
istioctl install --set profile=demo -y

# Enable sidecar injection
kubectl label namespace sage-system istio-injection=enabled

# Apply Gateway and VirtualService
kubectl apply -f ingress.yaml
```

### AWS ALB

```bash
# Install AWS Load Balancer Controller
eksctl utils associate-iam-oidc-provider --cluster=my-cluster --approve

kubectl apply -k "github.com/aws/eks-charts/stable/aws-load-balancer-controller//crds?ref=master"

helm repo add eks https://aws.github.io/eks-charts
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=my-cluster

# Apply ALB Ingress
kubectl apply -f ingress.yaml
```

## Monitoring

### Prometheus

```bash
# Install Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack -n observability --create-namespace

# Verify metrics endpoint
kubectl port-forward svc/sage-agent-metrics 9090:9090 -n sage-system
curl http://localhost:9090/metrics
```

### Grafana Dashboards

```bash
# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n observability

# Default credentials: admin / prom-operator

# Import dashboards from dashboards/
```

### Logging (ELK/Loki)

**Option 1: Loki (recommended for K8s)**
```bash
# Install Loki Stack
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n observability --set promtail.enabled=true

# Query logs in Grafana
```

**Option 2: ELK Stack**
```bash
# Install ECK (Elastic Cloud on Kubernetes)
kubectl create -f https://download.elastic.co/downloads/eck/2.9.0/crds.yaml
kubectl apply -f https://download.elastic.co/downloads/eck/2.9.0/operator.yaml

# Deploy Elasticsearch, Kibana, Filebeat
kubectl apply -f elk-stack.yaml
```

### Distributed Tracing (Jaeger)

```bash
# Install Jaeger Operator
kubectl create ns observability
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.48.0/jaeger-operator.yaml -n observability

# Deploy Jaeger
kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger
  namespace: observability
spec:
  strategy: production
  storage:
    type: elasticsearch
EOF

# Access Jaeger UI
kubectl port-forward svc/jaeger-query 16686:16686 -n observability
```

## High Availability

### Pod Disruption Budget

Prevent too many pods from being unavailable during maintenance:

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: sage-agent-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: sage-agent
```

```bash
kubectl apply -f pdb.yaml
```

### Anti-Affinity

Spread pods across nodes (already in `deployment.yaml`):

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - sage-agent
        topologyKey: kubernetes.io/hostname
```

### Multi-Zone Deployment

```yaml
# deployment.yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - sage-agent
      topologyKey: topology.kubernetes.io/zone
```

## Security

### Network Policies

Restrict pod-to-pod communication:

```yaml
# networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: sage-agent-netpol
spec:
  podSelector:
    matchLabels:
      app: sage-agent
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
    - namespaceSelector:
        matchLabels:
          name: sage-system
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
```

### RBAC

```yaml
# rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sage-agent
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sage-agent-role
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sage-agent-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: sage-agent-role
subjects:
- kind: ServiceAccount
  name: sage-agent
```

### Pod Security Standards

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: sage-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## Troubleshooting

### Common Issues

#### Pods in CrashLoopBackOff

```bash
# Check logs
kubectl logs -f pod/<pod-name> -n sage-system

# Describe pod
kubectl describe pod/<pod-name> -n sage-system

# Common causes:
# - Missing secrets
# - Database connection failure
# - Invalid configuration
```

#### Service Unreachable

```bash
# Check service
kubectl get svc sage-agent -n sage-system

# Check endpoints
kubectl get endpoints sage-agent -n sage-system

# Test connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- curl http://sage-agent.sage-system.svc.cluster.local:8080/health/ready
```

#### HPA Not Scaling

```bash
# Check HPA status
kubectl describe hpa sage-agent-hpa -n sage-system

# Check metrics server
kubectl top pods -n sage-system

# Install metrics-server if missing
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

### Debug Commands

```bash
# Get all resources
kubectl get all -n sage-system

# Check events
kubectl get events -n sage-system --sort-by='.lastTimestamp'

# Exec into pod
kubectl exec -it deployment/sage-agent -n sage-system -- /bin/sh

# Port forward for local testing
kubectl port-forward svc/sage-agent 8080:80 -n sage-system

# View resource usage
kubectl top pods -n sage-system
kubectl top nodes
```

## Backup & Disaster Recovery

### Backup Configuration

```bash
# Backup all manifests
kubectl get all,configmap,secret,ingress,hpa,pdb -n sage-system -o yaml > backup.yaml

# Backup using Velero
velero backup create sage-agent-backup --include-namespaces sage-system
```

### Disaster Recovery

```bash
# Restore from backup
kubectl apply -f backup.yaml

# Restore using Velero
velero restore create --from-backup sage-agent-backup
```

## Best Practices

1. **Resource Limits**: Always set CPU/memory requests and limits
2. **Health Checks**: Configure liveness, readiness, and startup probes
3. **Secrets**: Use external secret management (Vault, AWS Secrets Manager)
4. **Monitoring**: Deploy Prometheus, Grafana, Jaeger
5. **Logging**: Aggregate logs with Loki or ELK
6. **Scaling**: Use HPA with custom metrics
7. **HA**: Deploy across multiple zones with PDB
8. **Security**: Implement Network Policies and RBAC
9. **Updates**: Use rolling updates with maxSurge/maxUnavailable
10. **Backups**: Regular backups with Velero

## Next Steps

- [Configure monitoring](../../monitoring/prometheus-setup.md)
- [Set up logging](../../monitoring/logging-setup.md)
- [Security hardening](../../security/tls-ssl.md)
- [Performance tuning](../../troubleshooting/performance.md)
