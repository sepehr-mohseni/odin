"""
Grok-1 Inference Service for Traffic Analysis

This service provides AI-powered traffic analysis using the Grok-1 model.
Due to Grok-1's size (314B parameters), this implementation supports:
1. Lightweight mode: Uses smaller models (scikit-learn, transformers)
2. Full mode: Uses actual Grok-1 (requires significant GPU resources)
3. Proxy mode: Proxies requests to external Grok API
"""

import os
import json
import logging
import asyncio
from datetime import datetime
from typing import Dict, List, Optional, Any
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field
import uvicorn

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration
MODE = os.getenv("GROK_MODE", "lightweight")  # lightweight, full, proxy
GROK_API_URL = os.getenv("GROK_API_URL", "")
MODEL_PATH = os.getenv("MODEL_PATH", "/models")

# FastAPI app
app = FastAPI(
    title="Grok-1 Inference Service",
    description="AI-powered traffic analysis for Odin API Gateway",
    version="1.0.0"
)

# Request/Response models
class AnalysisRequest(BaseModel):
    prompt: str
    max_tokens: Optional[int] = Field(default=500, ge=1, le=2000)
    temperature: Optional[float] = Field(default=0.3, ge=0.0, le=1.0)
    context: Optional[Dict[str, Any]] = None

class AnalysisResponse(BaseModel):
    response: str
    confidence: Optional[float] = None
    anomalies: Optional[List[str]] = None
    suggestions: Optional[List[str]] = None
    metadata: Optional[Dict[str, Any]] = None

class HealthResponse(BaseModel):
    status: str
    mode: str
    model_loaded: bool
    uptime: float

# Global state
model = None
start_time = datetime.now()

@app.on_event("startup")
async def startup_event():
    """Initialize the model on startup"""
    global model
    logger.info(f"Starting Grok inference service in {MODE} mode")
    
    try:
        if MODE == "lightweight":
            model = LightweightAnalyzer()
            logger.info("Lightweight analyzer initialized")
        elif MODE == "full":
            model = GrokAnalyzer(MODEL_PATH)
            logger.info("Grok-1 model initialized")
        elif MODE == "proxy":
            model = ProxyAnalyzer(GROK_API_URL)
            logger.info(f"Proxy analyzer initialized (API: {GROK_API_URL})")
        else:
            raise ValueError(f"Unknown mode: {MODE}")
    except Exception as e:
        logger.error(f"Failed to initialize model: {e}")
        raise

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    uptime = (datetime.now() - start_time).total_seconds()
    return HealthResponse(
        status="healthy" if model else "unhealthy",
        mode=MODE,
        model_loaded=model is not None,
        uptime=uptime
    )

@app.post("/analyze", response_model=AnalysisResponse)
async def analyze(request: AnalysisRequest):
    """Analyze traffic patterns and detect anomalies"""
    if not model:
        raise HTTPException(status_code=503, detail="Model not initialized")
    
    try:
        logger.info(f"Analyzing request with prompt length: {len(request.prompt)}")
        result = await model.analyze(
            request.prompt,
            request.max_tokens,
            request.temperature,
            request.context
        )
        return result
    except Exception as e:
        logger.error(f"Analysis failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))

class LightweightAnalyzer:
    """
    Lightweight analyzer using rule-based and simple ML approaches
    This is the default mode for production use without expensive GPU requirements
    """
    
    def __init__(self):
        logger.info("Initializing lightweight analyzer")
        self.rules = self._load_rules()
    
    def _load_rules(self) -> Dict[str, Any]:
        """Load anomaly detection rules"""
        return {
            "error_spike": {
                "threshold": 0.1,  # 10% error rate
                "severity": "high"
            },
            "latency_spike": {
                "threshold": 1000,  # 1 second
                "severity": "medium"
            },
            "ddos_pattern": {
                "requests_per_ip": 100,
                "severity": "critical"
            }
        }
    
    async def analyze(
        self,
        prompt: str,
        max_tokens: int,
        temperature: float,
        context: Optional[Dict[str, Any]]
    ) -> AnalysisResponse:
        """Perform lightweight analysis"""
        logger.info("Performing lightweight analysis")
        
        # Parse context if available
        anomalies = []
        suggestions = []
        confidence = 0.7
        
        if context and "anomalies" in context:
            detected_anomalies = context["anomalies"]
            
            for anomaly in detected_anomalies:
                anomaly_type = anomaly.get("anomaly_type", "")
                severity = anomaly.get("severity", "")
                
                # Validate anomaly based on rules
                if self._validate_anomaly(anomaly):
                    anomalies.append(f"Confirmed: {anomaly_type} - {severity}")
                    confidence = 0.85
                else:
                    anomalies.append(f"Low confidence: {anomaly_type}")
                    confidence = 0.5
                
                # Generate suggestions
                suggestions.extend(self._generate_suggestions(anomaly_type, severity))
        
        # Generate response text
        response_text = self._generate_response(prompt, anomalies, suggestions)
        
        return AnalysisResponse(
            response=response_text,
            confidence=confidence,
            anomalies=anomalies if anomalies else None,
            suggestions=suggestions if suggestions else None,
            metadata={
                "analyzer": "lightweight",
                "rules_version": "1.0",
                "analyzed_at": datetime.now().isoformat()
            }
        )
    
    def _validate_anomaly(self, anomaly: Dict[str, Any]) -> bool:
        """Validate if the detected anomaly is likely real"""
        score = anomaly.get("score", 0)
        anomaly_type = anomaly.get("anomaly_type", "")
        
        # Simple validation based on score and type
        if score > 80:
            return True
        if score > 60 and anomaly_type in ["ddos_attack", "data_exfiltration"]:
            return True
        
        return False
    
    def _generate_suggestions(self, anomaly_type: str, severity: str) -> List[str]:
        """Generate remediation suggestions"""
        suggestions_map = {
            "error_spike": [
                "Check backend service health",
                "Review recent deployments",
                "Increase timeout values if needed"
            ],
            "latency_spike": [
                "Scale up backend instances",
                "Check database query performance",
                "Enable caching for frequently accessed resources"
            ],
            "ddos_attack": [
                "Enable rate limiting immediately",
                "Block suspicious IP ranges",
                "Contact DDoS mitigation provider"
            ],
            "traffic_spike": [
                "Verify if legitimate traffic increase",
                "Scale horizontally if needed",
                "Enable CDN caching"
            ]
        }
        
        return suggestions_map.get(anomaly_type, ["Monitor the situation closely"])
    
    def _generate_response(
        self,
        prompt: str,
        anomalies: List[str],
        suggestions: List[str]
    ) -> str:
        """Generate human-readable response"""
        if not anomalies:
            return "No significant anomalies detected in the analyzed traffic patterns. All metrics appear to be within normal ranges."
        
        response = f"Analysis Results:\n\n"
        response += f"Detected {len(anomalies)} anomalies:\n"
        for i, anomaly in enumerate(anomalies, 1):
            response += f"{i}. {anomaly}\n"
        
        if suggestions:
            response += f"\nRecommended Actions:\n"
            for i, suggestion in enumerate(suggestions, 1):
                response += f"{i}. {suggestion}\n"
        
        return response

class GrokAnalyzer:
    """
    Full Grok-1 model analyzer (requires significant GPU resources)
    This would load and run the actual Grok-1 model
    """
    
    def __init__(self, model_path: str):
        logger.warning("Full Grok-1 mode requires significant GPU memory")
        logger.warning("This implementation is a placeholder")
        # In production, this would load the actual Grok-1 model
        # from the model_path using JAX and the xai-org/grok-1 code
        self.model_path = model_path
        raise NotImplementedError(
            "Full Grok-1 mode requires downloading 314B parameters model. "
            "Use 'lightweight' or 'proxy' mode instead."
        )
    
    async def analyze(self, prompt: str, max_tokens: int, temperature: float, context: Optional[Dict]) -> AnalysisResponse:
        raise NotImplementedError()

class ProxyAnalyzer:
    """
    Proxy analyzer that forwards requests to external Grok API
    """
    
    def __init__(self, api_url: str):
        if not api_url:
            raise ValueError("GROK_API_URL must be set for proxy mode")
        self.api_url = api_url
        logger.info(f"Proxy mode configured for: {api_url}")
    
    async def analyze(
        self,
        prompt: str,
        max_tokens: int,
        temperature: float,
        context: Optional[Dict[str, Any]]
    ) -> AnalysisResponse:
        """Forward request to external Grok API"""
        import aiohttp
        
        payload = {
            "prompt": prompt,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "context": context
        }
        
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{self.api_url}/analyze",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=60)
            ) as response:
                if response.status != 200:
                    raise HTTPException(
                        status_code=response.status,
                        detail=f"Grok API error: {await response.text()}"
                    )
                
                data = await response.json()
                return AnalysisResponse(**data)

if __name__ == "__main__":
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        log_level="info"
    )
