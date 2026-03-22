"""Natural language to test case converter."""

from __future__ import annotations

import json
import logging

from probex_brain.ai.prompts import NL_TO_TEST_SYSTEM, NL_TO_TEST_USER_TEMPLATE
from probex_brain.ai.router import AIRouter
from probex_brain.models.schemas import (
    Assertion,
    NLTestRequest,
    NLTestResponse,
    TestCase,
    TestRequest,
)

logger = logging.getLogger(__name__)


class NLTestGenerator:
    """Converts natural language descriptions into executable API test cases."""

    def __init__(self, router: AIRouter) -> None:
        self.router = router

    async def generate(self, request: NLTestRequest) -> NLTestResponse:
        endpoints_json = json.dumps(
            [ep.model_dump(exclude_none=True) for ep in request.endpoints],
            indent=2,
        )

        context_section = ""
        if request.context:
            context_section = f"Additional context:\n{request.context}\n"

        prompt = NL_TO_TEST_USER_TEMPLATE.format(
            description=request.description,
            context=context_section,
            endpoints_json=endpoints_json,
        )

        data, _model_used = await self.router.generate_json(
            prompt, system=NL_TO_TEST_SYSTEM, max_tokens=8192
        )

        test_cases = _parse_test_cases(data)
        interpretation = data.get("interpretation", "")
        return NLTestResponse(test_cases=test_cases, interpretation=interpretation)


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
                    name=raw.get("name", "unnamed_nl_test"),
                    description=raw.get("description", ""),
                    category=raw.get("category", "functional"),
                    severity=raw.get("severity", "medium"),
                    request=test_request,
                    assertions=assertions,
                    tags=raw.get("tags", ["nl-generated"]),
                )
            )
        except Exception as exc:
            logger.warning("Skipping malformed NL test case: %s", exc)
    return cases
