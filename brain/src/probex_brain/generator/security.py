"""AI-powered security analyzer for API endpoints."""

from __future__ import annotations

import json
import logging

from probex_brain.ai.prompts import SECURITY_SYSTEM, SECURITY_USER_TEMPLATE
from probex_brain.ai.router import AIRouter
from probex_brain.models.schemas import (
    Assertion,
    SecurityAnalysisRequest,
    SecurityAnalysisResponse,
    SecurityFinding,
    TestCase,
    TestRequest,
)

logger = logging.getLogger(__name__)


class SecurityAnalyzer:
    """Analyzes API endpoints for security vulnerabilities using AI."""

    def __init__(self, router: AIRouter) -> None:
        self.router = router

    async def analyze(self, request: SecurityAnalysisRequest) -> SecurityAnalysisResponse:
        endpoint_json = json.dumps(
            request.endpoint.model_dump(exclude_none=True),
            indent=2,
        )

        context_section = ""
        if request.context:
            context_section = f"Additional context:\n{request.context}\n"

        prompt = SECURITY_USER_TEMPLATE.format(
            context=context_section,
            endpoint_json=endpoint_json,
        )

        data, _model_used = await self.router.generate_json(
            prompt, system=SECURITY_SYSTEM, max_tokens=8192
        )

        findings = _parse_findings(data)
        test_cases = _parse_test_cases(data)
        return SecurityAnalysisResponse(findings=findings, test_cases=test_cases)


def _parse_findings(data: dict) -> list[SecurityFinding]:
    raw_findings = data.get("findings", [])
    findings: list[SecurityFinding] = []
    for raw in raw_findings:
        try:
            findings.append(
                SecurityFinding(
                    title=raw.get("title", ""),
                    description=raw.get("description", ""),
                    severity=raw.get("severity", "medium"),
                    owasp_category=raw.get("owasp_category", ""),
                    recommendation=raw.get("recommendation", ""),
                )
            )
        except Exception as exc:
            logger.warning("Skipping malformed finding: %s", exc)
    return findings


def _parse_test_cases(data: dict) -> list[TestCase]:
    raw_cases = data.get("test_cases", [])
    cases: list[TestCase] = []
    for raw in raw_cases:
        try:
            req = raw.get("request", {})
            test_request = TestRequest(
                method=req.get("method", "GET"),
                url=req.get("url", ""),
                headers=req.get("headers", {}),
                body=req.get("body", ""),
                timeout=req.get("timeout", 30),
            )
            assertions = [
                Assertion(
                    type=a.get("type", "status_code"),
                    target=a.get("target", ""),
                    operator=a.get("operator", "eq"),
                    expected=str(a.get("expected", "")),
                )
                for a in raw.get("assertions", [])
            ]
            cases.append(
                TestCase(
                    name=raw.get("name", "unnamed_security_test"),
                    description=raw.get("description", ""),
                    category=raw.get("category", "security"),
                    severity=raw.get("severity", "high"),
                    request=test_request,
                    assertions=assertions,
                    tags=raw.get("tags", ["security"]),
                )
            )
        except Exception as exc:
            logger.warning("Skipping malformed security test case: %s", exc)
    return cases
