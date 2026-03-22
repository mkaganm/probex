"""AI Router — selects the best available provider based on configuration."""

from __future__ import annotations

import logging

from probex_brain.ai.base import AIProvider
from probex_brain.ai.cloud_provider import AnthropicProvider
from probex_brain.ai.local_provider import OllamaProvider
from probex_brain.models.config import AIConfig

logger = logging.getLogger(__name__)


class AIRouter:
    """Routes AI requests to the appropriate provider based on config and availability."""

    def __init__(self, config: AIConfig) -> None:
        self.config = config
        self.local: AIProvider | None = None
        self.cloud: AIProvider | None = None
        self._active_provider: AIProvider | None = None

    async def initialize(self) -> None:
        """Set up providers based on the configured mode."""
        if self.config.mode in ("local", "hybrid"):
            self.local = OllamaProvider(model=self.config.local_model)
            if await self.local.is_available():
                logger.info("Local provider ready: %s", self.local.name())
            else:
                logger.warning("Local provider unavailable — Ollama not reachable")
                if self.config.mode == "local":
                    logger.error("Mode is 'local' but Ollama is not available")

        if self.config.mode in ("cloud", "hybrid"):
            self.cloud = AnthropicProvider(
                api_key=self.config.cloud_api_key,
                model=self.config.cloud_model,
            )
            if await self.cloud.is_available():
                logger.info("Cloud provider ready: %s", self.cloud.name())
            else:
                logger.warning("Cloud provider unavailable — no API key set")

    def _ordered_providers(self) -> list[AIProvider]:
        """Return providers in preferred order."""
        providers: list[AIProvider] = []
        if self.config.mode == "offline":
            return providers
        if self.config.prefer_local:
            if self.local:
                providers.append(self.local)
            if self.cloud:
                providers.append(self.cloud)
        else:
            if self.cloud:
                providers.append(self.cloud)
            if self.local:
                providers.append(self.local)
        return providers

    async def generate(
        self, prompt: str, system: str = "", max_tokens: int = 4096
    ) -> tuple[str, str]:
        """Generate text. Returns (response_text, model_used)."""
        providers = self._ordered_providers()
        if not providers:
            raise RuntimeError(
                f"No AI providers available (mode={self.config.mode}). "
                "Set PROBEX_AI_MODE to 'local', 'cloud', or 'hybrid'."
            )

        last_error: Exception | None = None
        for provider in providers:
            try:
                if not await provider.is_available():
                    continue
                result = await provider.generate(prompt, system=system, max_tokens=max_tokens)
                self._active_provider = provider
                return result, provider.name()
            except Exception as exc:
                logger.warning("Provider %s failed: %s", provider.name(), exc)
                last_error = exc

        raise RuntimeError(
            f"All AI providers failed. Last error: {last_error}"
        )

    async def generate_json(
        self, prompt: str, system: str = "", max_tokens: int = 4096
    ) -> tuple[dict, str]:
        """Generate JSON. Returns (parsed_dict, model_used)."""
        providers = self._ordered_providers()
        if not providers:
            raise RuntimeError(
                f"No AI providers available (mode={self.config.mode}). "
                "Set PROBEX_AI_MODE to 'local', 'cloud', or 'hybrid'."
            )

        last_error: Exception | None = None
        for provider in providers:
            try:
                if not await provider.is_available():
                    continue
                result = await provider.generate_json(prompt, system=system, max_tokens=max_tokens)
                self._active_provider = provider
                return result, provider.name()
            except Exception as exc:
                logger.warning("Provider %s JSON generation failed: %s", provider.name(), exc)
                last_error = exc

        raise RuntimeError(
            f"All AI providers failed for JSON generation. Last error: {last_error}"
        )

    @property
    def active_model_name(self) -> str:
        if self._active_provider:
            return self._active_provider.name()
        return "none"

    @property
    def total_tokens(self) -> int:
        total = 0
        if isinstance(self.cloud, AnthropicProvider):
            total += self.cloud.total_input_tokens + self.cloud.total_output_tokens
        return total
