# Grok-1 Inference Service for Odin API Gateway
# 
# NOTE: The full Grok-1 model requires significant GPU memory (multiple A100s).
# This Docker image provides two modes:
# 1. Lightweight mode: Uses a smaller ML model for basic anomaly detection
# 2. Full mode: Downloads and runs Grok-1 (requires significant resources)

FROM python:3.11-slim as base

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    wget \
    curl \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY grok-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY grok-service/ .

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# Run the service
CMD ["python", "main.py"]
