"""Anthropic cloud AI provider with budget tracking."""

from __future__ import annotations

import json
import logging

from probex_brain.ai.base import AIProvider

logger = logging.getLogger(__name__)

# Approximate cost per million tokens (USD) for common Anthropic models.
_COST_PER_M_INPUT: dict[str, float] = {
    "claude-sonnet-4-6": 3.0,
    "claude-haiku-4": 0.80,
}
_COST_PER_M_OUTPUT: dict[str, float] = {
    "claude-sonnet-4-6": 15.0,
    "claude-haiku-4": 4.0,
}


class AnthropicProvider(AIProvider):
    """AI provider backed by the Anthropic Messages API."""

    def __init__(self, api_key: str, model: str = "claude-sonnet-4-6") -> None:
        self.model = model
        self._api_key = api_key
        self._client = None
        self.total_input_tokens: int = 0
        self.total_output_tokens: int = 0

    def _get_client(self):
        if self._client is None:
            import anthropic

            self._client = anthropic.AsyncAnthropic(api_key=self._api_key)
        return self._client

    @property
    def estimated_cost_usd(self) -> float:
        """Return estimated total spend so far in USD."""
        input_cost = (
            self.total_input_tokens / 1_000_000 * _COST_PER_M_INPUT.get(self.model, 3.0)
        )
        output_cost = (
            self.total_output_tokens / 1_000_000 * _COST_PER_M_OUTPUT.get(self.model, 15.0)
        )
        return input_cost + output_cost

    def _track_usage(self, usage) -> int:
        """Track token usage from an API response. Returns total tokens."""
        input_tokens = getattr(usage, "input_tokens", 0)
        output_tokens = getattr(usage, "output_tokens", 0)
        self.total_input_tokens += input_tokens
        self.total_output_tokens += output_tokens
        return input_tokens + output_tokens

    async def generate(self, prompt: str, system: str = "", max_tokens: int = 4096) -> str:
        client = self._get_client()
        kwargs: dict = {
            "model": self.model,
            "max_tokens": max_tokens,
            "messages": [{"role": "user", "content": prompt}],
        }
        if system:
            kwargs["system"] = system

        try:
            response = await client.messages.create(**kwargs)
            self._track_usage(response.usage)
            return response.content[0].text
        except Exception as exc:
            logger.error("Anthropic generate failed: %s", exc)
            raise

    async def generate_json(
        self, prompt: str, system: str = "", max_tokens: int = 4096
    ) -> dict:
        json_instruction = (
            "\n\nIMPORTANT: Respond with valid JSON only. "
            "Do not include markdown fences or any text outside the JSON object."
        )
        text = await self.generate(prompt + json_instruction, system=system, max_tokens=max_tokens)

        # Strip markdown fences if present.
        text = text.strip()
        if text.startswith("```"):
            first_newline = text.index("\n")
            text = text[first_newline + 1 :]
        if text.endswith("```"):
            text = text[: text.rfind("```")]
        text = text.strip()

        try:
            return json.loads(text)
        except json.JSONDecodeError as exc:
            logger.error("Anthropic returned invalid JSON: %s — raw: %.200s", exc, text)
            raise

    def name(self) -> str:
        return f"anthropic/{self.model}"

    async def is_available(self) -> bool:
        return bool(self._api_key)
