# Grok-1 Inference Service

AI-powered traffic analysis service for Odin API Gateway using Grok-1 model.

## Overview

This service provides anomaly detection and traffic analysis capabilities. Due to Grok-1's massive size (314B parameters), it supports three operating modes:

### Operating Modes

1. **Lightweight Mode** (Recommended for Production)
   - Uses rule-based and simple ML models
   - Low resource requirements (1-2 GB RAM, CPU only)
   - Fast inference (<100ms)
   - Good for real-time anomaly detection
   
2. **Full Mode** (Experimental)
   - Uses actual Grok-1 model (314B parameters)
   - Requires: Multiple A100 GPUs (80GB each), 1TB+ RAM
   - Slow inference (seconds to minutes)
   - Best accuracy for complex pattern analysis
   
3. **Proxy Mode**
   - Forwards requests to external Grok API
   - Minimal resource requirements
   - Depends on external service availability

## Quick Start

### Lightweight Mode (Default)

```bash
cd deployments/grok
docker build -t odin-grok:latest .
docker run -d \
  -p 8000:8000 \
  -e GROK_MODE=lightweight \
  --name odin-grok \
  odin-grok:latest
```

### Proxy Mode

```bash
docker run -d \
  -p 8000:8000 \
  -e GROK_MODE=proxy \
  -e GROK_API_URL=https://your-grok-api.com \
  --name odin-grok \
  odin-grok:latest
```

### Full Mode (Requires GPUs)

```bash
# Download Grok-1 model first (requires ~300GB)
mkdir -p /data/grok-models
cd /data/grok-models
# Use torrent or HuggingFace to download model

docker run -d \
  --gpus all \
  -p 8000:8000 \
  -v /data/grok-models:/models \
  -e GROK_MODE=full \
  -e MODEL_PATH=/models \
  --name odin-grok \
  odin-grok:latest
```

## API Endpoints

### Health Check

```bash
curl http://localhost:8000/health
```

Response:
```json
{
  "status": "healthy",
  "mode": "lightweight",
  "model_loaded": true,
  "uptime": 3600.5
}
```

### Analyze Traffic

```bash
curl -X POST http://localhost:8000/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Analyze these traffic anomalies",
    "max_tokens": 500,
    "temperature": 0.3,
    "context": {
      "anomalies": [
        {
          "anomaly_type": "error_spike",
          "severity": "high",
          "score": 85.5
        }
      ]
    }
  }'
```

Response:
```json
{
  "response": "Analysis Results:\n\nDetected 1 anomalies:\n1. Confirmed: error_spike - high\n\nRecommended Actions:\n1. Check backend service health\n2. Review recent deployments\n3. Increase timeout values if needed",
  "confidence": 0.85,
  "anomalies": ["Confirmed: error_spike - high"],
  "suggestions": [
    "Check backend service health",
    "Review recent deployments",
    "Increase timeout values if needed"
  ],
  "metadata": {
    "analyzer": "lightweight",
    "rules_version": "1.0",
    "analyzed_at": "2025-10-16T12:00:00"
  }
}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GROK_MODE` | `lightweight` | Operating mode: lightweight, full, proxy |
| `GROK_API_URL` | `""` | External Grok API URL (proxy mode only) |
| `MODEL_PATH` | `/models` | Path to Grok-1 model files (full mode only) |
| `PORT` | `8000` | Service port |

## Resource Requirements

### Lightweight Mode
- CPU: 2 cores
- RAM: 2 GB
- Disk: 1 GB
- GPU: Not required

### Full Mode
- CPU: 16+ cores
- RAM: 1 TB+
- Disk: 500 GB (for model storage)
- GPU: 4-8x NVIDIA A100 (80GB)

### Proxy Mode
- CPU: 1 core
- RAM: 512 MB
- Disk: 500 MB
- GPU: Not required

## Integration with Odin

In your Odin configuration:

```yaml
ai:
  enabled: true
  use_grok_model: true
  grok_service_url: "http://localhost:8000"
  grok_timeout: "30s"
  analysis_interval: "5m"
  anomaly_threshold: 3.0
```

## Development

### Running Locally

```bash
cd deployments/grok/grok-service
pip install -r requirements.txt
export GROK_MODE=lightweight
python main.py
```

### Testing

```bash
# Health check
curl http://localhost:8000/health

# Test analysis
curl -X POST http://localhost:8000/analyze \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Test analysis", "max_tokens": 100}'
```

## Production Deployment

### Docker Compose

See `docker-compose.yml` in the root deployments directory.

### Kubernetes

Helm chart includes Grok service deployment. For lightweight mode:

```yaml
grok:
  enabled: true
  mode: lightweight
  replicas: 2
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
```

## Troubleshooting

### Service Not Starting

Check logs:
```bash
docker logs odin-grok
```

Common issues:
- **Port already in use**: Change port with `-p 8001:8000`
- **Out of memory**: Ensure sufficient RAM (2GB+ for lightweight)
- **Model not found**: Check MODEL_PATH for full mode

### Slow Response Times

- **Lightweight mode**: Should respond in <100ms
- **Full mode**: May take seconds to minutes
- **Proxy mode**: Depends on external API

Solutions:
- Use lightweight mode for production
- Increase replicas for load distribution
- Enable response caching in Odin

### Low Confidence Scores

The lightweight analyzer uses rule-based detection. Low confidence (<0.7) means:
- Anomaly score is borderline
- Pattern doesn't match known rules
- More historical data needed for baseline

Consider:
- Adjusting `anomaly_threshold` in Odin config
- Training period for baseline establishment
- Upgrading to full mode for better accuracy

## License

Apache 2.0 (same as Grok-1 open release)
