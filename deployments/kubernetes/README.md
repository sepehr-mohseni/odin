# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying Odin API Gateway.

## Files

- `deployment.yaml`: Kubernetes Deployment resource
- `service.yaml`: Kubernetes Service resource
- `ingress.yaml`: Kubernetes Ingress resource for external access
- `configmap.yaml`: ConfigMap for non-sensitive configuration
- `secrets.yaml`: Template for creating required Secrets

## Usage

To deploy to Kubernetes:

```bash
kubectl apply -f configmap.yaml
# Create secrets manually or using a secure method
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
```

For Helm-based deployment, use the chart in the `helm/` directory.
