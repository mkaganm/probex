"""FastAPI server for the PROBEX AI brain service."""

from __future__ import annotations

import argparse
import logging
import os
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, HTTPException

from probex_brain import __version__
from probex_brain.ai.router import AIRouter
from probex_brain.analysis.anomaly import AnomalyClassifier
from probex_brain.generator.nl_to_test import NLTestGenerator
from probex_brain.generator.scenario import ScenarioGenerator
from probex_brain.generator.security import SecurityAnalyzer
from probex_brain.models.config import AIConfig, ServerConfig
from probex_brain.models.schemas import (
    AnomalyClassifyRequest,
    AnomalyClassifyResponse,
    HealthResponse,
    NLTestRequest,
    NLTestResponse,
    ScenarioRequest,
    ScenarioResponse,
    SecurityAnalysisRequest,
    SecurityAnalysisResponse,
)

logger = logging.getLogger("probex_brain")

# Module-level state populated during lifespan.
_router: AIRouter | None = None
_ai_config: AIConfig | None = None


def _load_config() -> tuple[AIConfig, ServerConfig]:
    """Load configuration from environment variables."""
    ai = AIConfig(
        mode=os.getenv("PROBEX_AI_MODE", "offline"),
        local_model=os.getenv("PROBEX_LOCAL_MODEL", "qwen3:4b"),
        cloud_model=os.getenv("PROBEX_CLOUD_MODEL", "claude-sonnet-4-6"),
        cloud_api_key=os.getenv("PROBEX_CLOUD_API_KEY", ""),
        max_monthly_cost=float(os.getenv("PROBEX_MAX_MONTHLY_COST", "20.0")),
        prefer_local=os.getenv("PROBEX_PREFER_LOCAL", "true").lower() == "true",
    )
    server = ServerConfig(
        host=os.getenv("PROBEX_HOST", "127.0.0.1"),
        port=int(os.getenv("PROBEX_PORT", "9711")),
        log_level=os.getenv("PROBEX_LOG_LEVEL", "info"),
    )
    return ai, server


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize the AI router on startup."""
    global _router, _ai_config
    ai_config, _server_config = _load_config()
    _ai_config = ai_config

    router = AIRouter(ai_config)
    await router.initialize()
    _router = router

    logger.info(
        "PROBEX Brain v%s started — mode=%s",
        __version__,
        ai_config.mode,
    )
    yield
    logger.info("PROBEX Brain shutting down")


app = FastAPI(
    title="PROBEX AI Brain",
    version=__version__,
    description="Intelligent API test generation service",
    lifespan=lifespan,
)


def _get_router() -> AIRouter:
    if _router is None:
        raise HTTPException(status_code=503, detail="AI router not initialized")
    return _router


# ---------------------------------------------------------------------------
# Endpoints
# ---------------------------------------------------------------------------


@app.post("/api/v1/scenarios/generate", response_model=ScenarioResponse)
async def generate_scenarios(request: ScenarioRequest) -> ScenarioResponse:
    router = _get_router()
    generator = ScenarioGenerator(router)
    return await generator.generate(request)


@app.post("/api/v1/security/analyze", response_model=SecurityAnalysisResponse)
async def analyze_security(request: SecurityAnalysisRequest) -> SecurityAnalysisResponse:
    router = _get_router()
    analyzer = SecurityAnalyzer(router)
    return await analyzer.analyze(request)


@app.post("/api/v1/nl-to-test", response_model=NLTestResponse)
async def nl_to_test(request: NLTestRequest) -> NLTestResponse:
    router = _get_router()
    generator = NLTestGenerator(router)
    return await generator.generate(request)


@app.post("/api/v1/anomaly/classify", response_model=AnomalyClassifyResponse)
async def classify_anomaly(request: AnomalyClassifyRequest) -> AnomalyClassifyResponse:
    router = _get_router()
    classifier = AnomalyClassifier(router)
    return await classifier.classify(request)


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    config = _ai_config or AIConfig()
    router = _router
    model = router.active_model_name if router else "none"
    return HealthResponse(
        status="ok",
        version=__version__,
        ai_mode=config.mode,
        model=model,
    )


# ---------------------------------------------------------------------------
# CLI entrypoint
# ---------------------------------------------------------------------------


def main() -> None:
    parser = argparse.ArgumentParser(description="PROBEX AI Brain server")
    parser.add_argument("--host", default=None, help="Bind host (default from env or 127.0.0.1)")
    parser.add_argument("--port", type=int, default=None, help="Bind port (default from env or 9711)")
    args = parser.parse_args()

    _ai_config_boot, server_config = _load_config()

    host = args.host or server_config.host
    port = args.port or server_config.port

    logging.basicConfig(
        level=getattr(logging, server_config.log_level.upper(), logging.INFO),
        format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
    )

    uvicorn.run(
        "probex_brain.server:app",
        host=host,
        port=port,
        log_level=server_config.log_level,
    )


if __name__ == "__main__":
    main()
