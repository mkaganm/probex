"""AI-powered test scenario generator."""

from __future__ import annotations

import json
import logging

from probex_brain.ai.prompts import SCENARIO_SYSTEM, SCENARIO_USER_TEMPLATE
from probex_brain.ai.router import AIRouter
from probex_brain.models.schemas import (
    Assertion,
    ScenarioRequest,
    ScenarioResponse,
    TestCase,
    TestRequest,
)

logger = logging.getLogger(__name__)


class ScenarioGenerator:
    """Generates test scenarios by prompting an AI model."""

    def __init__(self, router: AIRouter) -> None:
        self.router = router

    async def generate(self, request: ScenarioRequest) -> ScenarioResponse:
        endpoints_json = json.dumps(
            [ep.model_dump(exclude_none=True) for ep in request.endpoints],
            indent=2,
        )

        profile_section = ""
        if request.profile_context:
            profile_section = f"Profile context:\n{request.profile_context}\n"

        prompt = SCENARIO_USER_TEMPLATE.format(
            max_scenarios=request.max_scenarios,
            profile_context=profile_section,
            endpoints_json=endpoints_json,
        )

        data, model_used = await self.router.generate_json(
            prompt, system=SCENARIO_SYSTEM, max_tokens=8192
        )

        scenarios = _parse_scenarios(data)
        return ScenarioResponse(
            scenarios=scenarios,
            model_used=model_used,
            tokens_used=self.router.total_tokens,
        )


def _parse_scenarios(data: dict) -> list[TestCase]:
    """Parse the AI response into validated TestCase objects."""
    raw_scenarios = data.get("scenarios", [])
    cases: list[TestCase] = []
    for raw in raw_scenarios:
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
            case = TestCase(
                name=raw.get("name", "unnamed_test"),
                description=raw.get("description", ""),
                category=raw.get("category", "functional"),
                severity=raw.get("severity", "medium"),
                request=test_request,
                assertions=assertions,
                tags=raw.get("tags", []),
            )
            cases.append(case)
        except Exception as exc:
            logger.warning("Skipping malformed scenario: %s", exc)
    return cases
