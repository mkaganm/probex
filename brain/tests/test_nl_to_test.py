"""Tests for probex_brain.generator.nl_to_test."""

from __future__ import annotations

import pytest

from probex_brain.generator.nl_to_test import NLTestGenerator, _parse_test_cases
from probex_brain.models.schemas import Endpoint, NLTestRequest

from .conftest import FakeAIProvider

VALID_NL_DATA = {
    "interpretation": "Testing user listing endpoint returns 200",
    "test_cases": [
        {
            "name": "list_users_ok",
            "description": "Verify GET /users returns 200",
            "request": {"method": "GET", "url": "/api/users"},
            "assertions": [{"type": "status_code", "expected": "200"}],
            "tags": ["nl-generated"],
        }
    ],
}


def test_parse_test_cases_valid():
    cases = _parse_test_cases(VALID_NL_DATA)
    assert len(cases) == 1
    assert cases[0].name == "list_users_ok"
    assert "nl-generated" in cases[0].tags
    assert cases[0].request.method == "GET"


def test_parse_empty_response():
    cases = _parse_test_cases({})
    assert cases == []


@pytest.mark.asyncio
async def test_generate_returns_interpretation(fake_router, fake_provider):
    fake_provider._json = VALID_NL_DATA

    generator = NLTestGenerator(fake_router)
    request = NLTestRequest(
        description="Test that listing users works",
        endpoints=[Endpoint(method="GET", path="/api/users")],
    )
    response = await generator.generate(request)

    assert response.interpretation == "Testing user listing endpoint returns 200"
    assert len(response.test_cases) == 1
    assert response.test_cases[0].name == "list_users_ok"
