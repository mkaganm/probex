"""Tests for probex_brain.generator.security."""

from __future__ import annotations

import pytest

from probex_brain.generator.security import SecurityAnalyzer, _parse_findings, _parse_test_cases
from probex_brain.models.schemas import Endpoint, SecurityAnalysisRequest

from .conftest import FakeAIProvider

VALID_SECURITY_DATA = {
    "findings": [
        {
            "title": "SQL Injection",
            "description": "User input not sanitized",
            "severity": "critical",
            "owasp_category": "A03:2021",
            "recommendation": "Use parameterized queries",
        }
    ],
    "test_cases": [
        {
            "name": "sqli_test",
            "description": "Inject SQL via query param",
            "category": "security",
            "severity": "high",
            "request": {"method": "GET", "url": "/api/users?id=1' OR '1'='1"},
            "assertions": [{"type": "status_code", "expected": "400"}],
            "tags": ["security"],
        }
    ],
}


def test_parse_findings_valid():
    findings = _parse_findings(VALID_SECURITY_DATA)
    assert len(findings) == 1
    assert findings[0].title == "SQL Injection"
    assert findings[0].severity == "critical"
    assert findings[0].owasp_category == "A03:2021"
    assert findings[0].recommendation == "Use parameterized queries"


def test_parse_test_cases_valid():
    cases = _parse_test_cases(VALID_SECURITY_DATA)
    assert len(cases) == 1
    assert cases[0].name == "sqli_test"
    assert cases[0].category == "security"
    assert cases[0].request.url == "/api/users?id=1' OR '1'='1"
    assert "security" in cases[0].tags


@pytest.mark.asyncio
async def test_analyze_calls_router(fake_router, fake_provider):
    fake_provider._json = VALID_SECURITY_DATA

    analyzer = SecurityAnalyzer(fake_router)
    request = SecurityAnalysisRequest(endpoint=Endpoint(method="GET", path="/api/users"))
    response = await analyzer.analyze(request)

    assert len(response.findings) == 1
    assert response.findings[0].title == "SQL Injection"
    assert len(response.test_cases) == 1
    assert response.test_cases[0].name == "sqli_test"
