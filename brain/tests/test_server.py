"""Tests for probex_brain.server FastAPI endpoints."""

from __future__ import annotations

import pytest
import pytest_asyncio
from httpx import ASGITransport, AsyncClient

import probex_brain.server as server_module
from probex_brain.models.config import AIConfig
from probex_brain.server import app

from .conftest import FakeAIProvider


@pytest_asyncio.fixture
async def client(fake_router, fake_provider):
    server_module._router = fake_router
    server_module._ai_config = AIConfig(mode="local")
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as c:
        yield c
    server_module._router = None
    server_module._ai_config = None


@pytest.mark.asyncio
async def test_health_endpoint(client):
    resp = await client.get("/health")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["ai_mode"] == "local"


@pytest.mark.asyncio
async def test_generate_scenarios(client, fake_provider):
    fake_provider._json = {
        "scenarios": [
            {
                "name": "test1",
                "request": {"method": "GET", "url": "/api/users"},
                "assertions": [{"type": "status_code", "expected": "200"}],
            }
        ]
    }
    resp = await client.post(
        "/api/v1/scenarios/generate",
        json={
            "endpoints": [{"method": "GET", "path": "/users"}],
            "max_scenarios": 5,
        },
    )
    assert resp.status_code == 200
    body = resp.json()
    assert len(body["scenarios"]) == 1
    assert body["scenarios"][0]["name"] == "test1"


@pytest.mark.asyncio
async def test_security_analyze(client, fake_provider):
    fake_provider._json = {
        "findings": [
            {
                "title": "XSS",
                "description": "Reflected XSS",
                "severity": "high",
            }
        ],
        "test_cases": [],
    }
    resp = await client.post(
        "/api/v1/security/analyze",
        json={"endpoint": {"method": "GET", "path": "/search"}},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert len(body["findings"]) == 1
    assert body["findings"][0]["title"] == "XSS"


@pytest.mark.asyncio
async def test_nl_to_test(client, fake_provider):
    fake_provider._json = {
        "interpretation": "Check users endpoint",
        "test_cases": [
            {
                "name": "nl_test1",
                "request": {"method": "GET", "url": "/api/users"},
                "assertions": [{"type": "status_code", "expected": "200"}],
                "tags": ["nl-generated"],
            }
        ],
    }
    resp = await client.post(
        "/api/v1/nl-to-test",
        json={
            "description": "Test listing users",
            "endpoints": [{"method": "GET", "path": "/api/users"}],
        },
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["interpretation"] == "Check users endpoint"
    assert len(body["test_cases"]) == 1


@pytest.mark.asyncio
async def test_anomaly_classify(client, fake_provider):
    fake_provider._json = {
        "classification": "spike",
        "severity": "high",
        "explanation": "Response time doubled",
        "recommended_action": "Investigate load",
    }
    resp = await client.post(
        "/api/v1/anomaly/classify",
        json={
            "endpoint_id": "/api/users",
            "metric": "response_time_ms",
            "expected": 100.0,
            "actual": 250.0,
            "z_score": 3.5,
        },
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["classification"] == "spike"
    assert body["severity"] == "high"


@pytest.mark.asyncio
async def test_router_not_initialized():
    server_module._router = None
    server_module._ai_config = None
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as c:
        resp = await c.post(
            "/api/v1/scenarios/generate",
            json={
                "endpoints": [{"method": "GET", "path": "/x"}],
            },
        )
    assert resp.status_code == 503
