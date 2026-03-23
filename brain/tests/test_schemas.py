"""Tests for PROBEX Brain Pydantic models — serialization and validation."""

from __future__ import annotations

import json

import pytest

from probex_brain.models.schemas import (
    AnomalyClassifyRequest,
    AnomalyClassifyResponse,
    Assertion,
    AuthInfo,
    Endpoint,
    HealthResponse,
    NLTestRequest,
    NLTestResponse,
    Parameter,
    Schema,
    ScenarioRequest,
    ScenarioResponse,
    SecurityAnalysisRequest,
    SecurityAnalysisResponse,
    SecurityFinding,
    TestCase,
    TestRequest,
)


# --- Primitive models ---


class TestParameter:
    def test_defaults(self):
        p = Parameter(name="page")
        assert p.name == "page"
        assert p.type == "string"
        assert p.required is False
        assert p.example == ""

    def test_full(self):
        p = Parameter(name="limit", type="integer", required=True, example="25")
        data = p.model_dump()
        assert data["required"] is True
        rebuilt = Parameter.model_validate(data)
        assert rebuilt == p


class TestSchema:
    def test_nested_schema(self):
        s = Schema(
            type="object",
            properties={
                "id": Schema(type="integer"),
                "tags": Schema(type="array", items=Schema(type="string")),
            },
            required=["id"],
        )
        data = s.model_dump(exclude_none=True)
        assert data["properties"]["tags"]["items"]["type"] == "string"

    def test_enum(self):
        s = Schema(type="string", enum=["active", "inactive"])
        assert s.enum == ["active", "inactive"]


class TestAuthInfo:
    def test_basic(self):
        a = AuthInfo(type="bearer", location="header", key="Authorization")
        assert a.model_dump()["type"] == "bearer"


# --- Endpoint ---


class TestEndpoint:
    def test_minimal(self):
        ep = Endpoint(method="GET", path="/users")
        assert ep.base_url == ""
        assert ep.query_params == []

    def test_full_roundtrip(self):
        ep = Endpoint(
            method="POST",
            path="/users",
            base_url="https://api.example.com",
            query_params=[Parameter(name="dry_run", type="boolean")],
            path_params=[],
            request_body=Schema(type="object", properties={"name": Schema(type="string")}),
            auth=AuthInfo(type="bearer", location="header", key="Authorization"),
            tags=["users", "admin"],
        )
        json_str = ep.model_dump_json()
        rebuilt = Endpoint.model_validate_json(json_str)
        assert rebuilt.method == "POST"
        assert rebuilt.tags == ["users", "admin"]


# --- Test models ---


class TestTestRequest:
    def test_defaults(self):
        r = TestRequest(method="GET", url="http://localhost/health")
        assert r.timeout == 30
        assert r.headers == {}


class TestAssertion:
    def test_creation(self):
        a = Assertion(type="status_code", operator="eq", expected="200")
        assert a.target == ""


class TestTestCase:
    def test_full(self):
        tc = TestCase(
            name="test_get_user",
            description="Fetch user by ID",
            category="functional",
            severity="medium",
            request=TestRequest(method="GET", url="http://localhost/users/1"),
            assertions=[
                Assertion(type="status_code", operator="eq", expected="200"),
                Assertion(type="body_json", target="$.id", operator="eq", expected="1"),
            ],
            tags=["users"],
        )
        data = tc.model_dump()
        assert len(data["assertions"]) == 2
        assert data["request"]["method"] == "GET"


# --- Scenario models ---


class TestScenarioRequest:
    def test_defaults(self):
        req = ScenarioRequest(endpoints=[Endpoint(method="GET", path="/ping")])
        assert req.max_scenarios == 10
        assert req.profile_context is None


class TestScenarioResponse:
    def test_empty(self):
        resp = ScenarioResponse(scenarios=[], model_used="test", tokens_used=0)
        assert resp.model_dump()["model_used"] == "test"


# --- Security models ---


class TestSecurityFinding:
    def test_creation(self):
        f = SecurityFinding(
            title="Missing auth",
            description="Endpoint has no authentication",
            severity="high",
            owasp_category="API2:2023",
            recommendation="Add bearer token validation",
        )
        assert f.severity == "high"


class TestSecurityAnalysis:
    def test_roundtrip(self):
        req = SecurityAnalysisRequest(
            endpoint=Endpoint(method="DELETE", path="/users/{id}"),
            context="Admin-only endpoint",
        )
        resp = SecurityAnalysisResponse(
            findings=[SecurityFinding(title="IDOR", description="No ownership check")],
            test_cases=[],
        )
        assert len(resp.findings) == 1
        assert req.context == "Admin-only endpoint"


# --- NL to Test models ---


class TestNLModels:
    def test_request(self):
        req = NLTestRequest(
            description="Test that creating a user returns 201",
            endpoints=[Endpoint(method="POST", path="/users")],
        )
        assert req.context is None

    def test_response(self):
        resp = NLTestResponse(
            test_cases=[],
            interpretation="User wants to test POST /users returns 201",
        )
        assert resp.interpretation != ""


# --- Anomaly models ---


class TestAnomalyModels:
    def test_request(self):
        req = AnomalyClassifyRequest(
            endpoint_id="GET:/users",
            metric="p99_latency_ms",
            expected=150.0,
            actual=890.0,
            z_score=4.2,
            description="Latency spike after deploy",
        )
        assert req.z_score == 4.2

    def test_response(self):
        resp = AnomalyClassifyResponse(
            classification="spike",
            severity="high",
            explanation="Sudden latency increase",
            recommended_action="Check recent deployments",
        )
        data = json.loads(resp.model_dump_json())
        assert data["classification"] == "spike"


# --- Health ---


class TestHealthResponse:
    def test_defaults(self):
        h = HealthResponse()
        assert h.status == "ok"
        assert h.version == ""

    def test_full(self):
        h = HealthResponse(
            status="ok",
            version="0.4.0",
            ai_mode="hybrid",
            model="ollama/qwen3:4b",
        )
        assert h.ai_mode == "hybrid"
