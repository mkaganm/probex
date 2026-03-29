"""Tests for probex_brain.ai.router.AIRouter."""

from __future__ import annotations

import pytest

from probex_brain.ai.router import AIRouter
from probex_brain.models.config import AIConfig

from .conftest import FakeAIProvider


@pytest.mark.asyncio
async def test_offline_mode_raises():
    config = AIConfig(mode="offline")
    router = AIRouter(config)
    with pytest.raises(RuntimeError, match="No AI providers available"):
        await router.generate("hello")


@pytest.mark.asyncio
async def test_local_mode_generate():
    provider = FakeAIProvider(text_response="local reply")
    config = AIConfig(mode="local")
    router = AIRouter(config)
    router.local = provider
    text, model = await router.generate("prompt")
    assert text == "local reply"
    assert model == "fake-provider"


@pytest.mark.asyncio
async def test_cloud_mode_generate_json():
    provider = FakeAIProvider(json_response={"key": "value"})
    config = AIConfig(mode="cloud", prefer_local=False)
    router = AIRouter(config)
    router.cloud = provider
    data, model = await router.generate_json("prompt")
    assert data == {"key": "value"}
    assert model == "fake-provider"


@pytest.mark.asyncio
async def test_hybrid_prefer_local_order():
    local = FakeAIProvider(text_response="from local")
    cloud = FakeAIProvider(text_response="from cloud")
    config = AIConfig(mode="hybrid", prefer_local=True)
    router = AIRouter(config)
    router.local = local
    router.cloud = cloud

    ordered = router._ordered_providers()
    assert ordered[0] is local
    assert ordered[1] is cloud

    text, model = await router.generate("prompt")
    assert text == "from local"


@pytest.mark.asyncio
async def test_fallback_on_failure():
    failing = FakeAIProvider(text_response="should not appear")
    failing._available = False
    healthy = FakeAIProvider(text_response="fallback reply")

    config = AIConfig(mode="hybrid", prefer_local=True)
    router = AIRouter(config)
    router.local = failing
    router.cloud = healthy

    text, model = await router.generate("prompt")
    assert text == "fallback reply"
    assert model == "fake-provider"


@pytest.mark.asyncio
async def test_active_model_name():
    config = AIConfig(mode="local")
    router = AIRouter(config)
    assert router.active_model_name == "none"

    provider = FakeAIProvider()
    router.local = provider
    await router.generate("prompt")
    assert router.active_model_name == "fake-provider"
