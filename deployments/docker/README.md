# Docker Deployment

This directory contains Docker-specific deployment configurations for Odin API Gateway.

## Files

- `docker-compose.prod.yml`: Production-ready Docker Compose configuration
- `Dockerfile.prod`: Production-optimized Dockerfile

## Usage

To build and run the production Docker image:

```bash
docker build -f Dockerfile.prod -t odin-api-gateway:latest .
docker run -p 8080:8080 -v $(pwd)/config:/app/config odin-api-gateway:latest
```

For Docker Compose deployment:

```bash
docker-compose -f docker-compose.prod.yml up -d
```
