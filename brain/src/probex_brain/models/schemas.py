"""Pydantic models matching the Go PROBEX models for REST communication."""

from __future__ import annotations

from pydantic import BaseModel, Field


# --- API Schema Models ---


class Parameter(BaseModel):
    name: str
    type: str = "string"
    required: bool = False
    example: str = ""


class Schema(BaseModel):
    type: str = "object"
    properties: dict[str, Schema] | None = None
    items: Schema | None = None
    required: list[str] | None = None
    enum: list[str] | None = None
    pattern: str = ""
    format: str = ""


class AuthInfo(BaseModel):
    type: str = ""  # bearer, api_key, basic, oauth2
    location: str = ""  # header, query, cookie
    key: str = ""


class Endpoint(BaseModel):
    method: str
    path: str
    base_url: str = ""
    query_params: list[Parameter] = Field(default_factory=list)
    path_params: list[Parameter] = Field(default_factory=list)
    request_body: Schema | None = None
    responses: dict[str, Schema] = Field(default_factory=dict)
    auth: AuthInfo | None = None
    tags: list[str] = Field(default_factory=list)


# --- Test Models ---


class TestRequest(BaseModel):
    method: str
    url: str
    headers: dict[str, str] = Field(default_factory=dict)
    body: str = ""
    timeout: int = 30


class Assertion(BaseModel):
    type: str  # status_code, header, body_json, body_contains, response_time
    target: str = ""
    operator: str = "eq"  # eq, ne, gt, lt, gte, lte, contains, matches
    expected: str = ""


class TestCase(BaseModel):
    name: str
    description: str = ""
    category: str = ""  # functional, security, edge_case, performance, scenario
    severity: str = "medium"  # low, medium, high, critical
    request: TestRequest
    assertions: list[Assertion] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)


# --- Scenario Generation ---


class ScenarioRequest(BaseModel):
    endpoints: list[Endpoint]
    profile_context: str | None = None
    max_scenarios: int = 10


class ScenarioResponse(BaseModel):
    scenarios: list[TestCase]
    model_used: str
    tokens_used: int = 0


# --- Security Analysis ---


class SecurityFinding(BaseModel):
    title: str
    description: str
    severity: str = "medium"  # low, medium, high, critical
    owasp_category: str = ""
    recommendation: str = ""


class SecurityAnalysisRequest(BaseModel):
    endpoint: Endpoint
    context: str | None = None


class SecurityAnalysisResponse(BaseModel):
    findings: list[SecurityFinding]
    test_cases: list[TestCase]


# --- Natural Language to Test ---


class NLTestRequest(BaseModel):
    description: str
    endpoints: list[Endpoint]
    context: str | None = None


class NLTestResponse(BaseModel):
    test_cases: list[TestCase]
    interpretation: str = ""


# --- Anomaly Classification ---


class AnomalyClassifyRequest(BaseModel):
    endpoint_id: str
    metric: str
    expected: float
    actual: float
    z_score: float = 0.0
    description: str = ""


class AnomalyClassifyResponse(BaseModel):
    classification: str  # regression, spike, degradation, flaky, normal
    severity: str = "medium"
    explanation: str = ""
    recommended_action: str = ""


# --- Health ---


class HealthResponse(BaseModel):
    status: str = "ok"
    version: str = ""
    ai_mode: str = "offline"
    model: str = ""
