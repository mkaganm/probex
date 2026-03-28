"""Shared fixtures for PROBEX brain tests."""

from __future__ import annotations

import pytest

from probex_brain.ai.base import AIProvider
from probex_brain.ai.router import AIRouter
from probex_brain.models.config import AIConfig


class FakeAIProvider(AIProvider):
    """In-memory AI provider for tests — no network calls."""

    def __init__(
        self,
        text_response: str = "fake response",
        json_response: dict | None = None,
    ) -> None:
        self._text = text_response
        self._json = json_response or {}
        self._available = True

    async def generate(self, prompt: str, system: str = "", max_tokens: int = 4096) -> str:
        return self._text

    async def generate_json(self, prompt: str, system: str = "", max_tokens: int = 4096) -> dict:
        return self._json

    def name(self) -> str:
        return "fake-provider"

    async def is_available(self) -> bool:
        return self._available


@pytest.fixture
def fake_provider() -> FakeAIProvider:
    return FakeAIProvider()


@pytest.fixture
def fake_router(fake_provider: FakeAIProvider) -> AIRouter:
    config = AIConfig(mode="local")
    router = AIRouter(config)
    router.local = fake_provider
    router._active_provider = fake_provider
    return router
