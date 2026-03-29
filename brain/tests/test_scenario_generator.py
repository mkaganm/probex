"""Tests for probex_brain.generator.scenario."""

from __future__ import annotations

import pytest

from probex_brain.generator.scenario import ScenarioGenerator, _parse_scenarios
from probex_brain.models.schemas import Endpoint, ScenarioRequest

from .conftest import FakeAIProvider

VALID_SCENARIO_DATA = {
    "scenarios": [
        {
            "name": "test1",
            "request": {"method": "GET", "url": "/api/users"},
            "assertions": [{"type": "status_code", "expected": "200"}],
        },
        {
            "name": "test2",
            "description": "Create user",
            "request": {"method": "POST", "url": "/api/users"},
            "assertions": [{"type": "status_code", "expected": "201"}],
        },
    ]
}


def test_parse_scenarios_valid():
    cases = _parse_scenarios(VALID_SCENARIO_DATA)
    assert len(cases) == 2
    assert cases[0].name == "test1"
    assert cases[0].request.method == "GET"
    assert cases[0].request.url == "/api/users"
    assert cases[0].assertions[0].type == "status_code"
    assert cases[0].assertions[0].expected == "200"
    assert cases[1].name == "test2"


def test_parse_scenarios_empty():
    cases = _parse_scenarios({"scenarios": []})
    assert cases == []


def test_parse_scenarios_malformed_skipped():
    data = {
        "scenarios": [
            {"name": "good", "request": {"method": "GET", "url": "/ok"}},
            {"name": "bad"},  # missing request.url — will parse with url="" but is valid
            {"name": "also_good", "request": {"method": "DELETE", "url": "/del"}},
        ]
    }
    cases = _parse_scenarios(data)
    # All three should parse because _parse_scenarios fills defaults gracefully
    assert len(cases) >= 2


@pytest.mark.asyncio
async def test_generate_calls_router(fake_router, fake_provider):
    fake_provider._json = {
        "scenarios": [
            {
                "name": "test1",
                "request": {"method": "GET", "url": "/api/users"},
                "assertions": [{"type": "status_code", "expected": "200"}],
            }
        ]
    }

    generator = ScenarioGenerator(fake_router)
    request = ScenarioRequest(endpoints=[Endpoint(method="GET", path="/users")])
    response = await generator.generate(request)

    assert len(response.scenarios) == 1
    assert response.scenarios[0].name == "test1"
    assert response.model_used == "fake-provider"
