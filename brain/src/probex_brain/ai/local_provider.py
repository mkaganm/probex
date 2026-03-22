"""Ollama-based local AI provider."""

from __future__ import annotations

import json
import logging

from probex_brain.ai.base import AIProvider

logger = logging.getLogger(__name__)


class OllamaProvider(AIProvider):
    """AI provider backed by a local Ollama instance."""

    def __init__(self, model: str = "qwen3:4b", host: str | None = None) -> None:
        self.model = model
        self._host = host
        self._client = None

    def _get_client(self):
        if self._client is None:
            import ollama

            kwargs = {}
            if self._host:
                kwargs["host"] = self._host
            self._client = ollama.AsyncClient(**kwargs)
        return self._client

    async def generate(self, prompt: str, system: str = "", max_tokens: int = 4096) -> str:
        client = self._get_client()
        messages = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        try:
            response = await client.chat(
                model=self.model,
                messages=messages,
                options={"num_predict": max_tokens},
            )
            return response["message"]["content"]
        except Exception as exc:
            logger.error("Ollama generate failed: %s", exc)
            raise

    async def generate_json(
        self, prompt: str, system: str = "", max_tokens: int = 4096
    ) -> dict:
        client = self._get_client()
        messages = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        try:
            response = await client.chat(
                model=self.model,
                messages=messages,
                format="json",
                options={"num_predict": max_tokens},
            )
            text = response["message"]["content"]
            return json.loads(text)
        except json.JSONDecodeError as exc:
            logger.error("Ollama returned invalid JSON: %s", exc)
            raise
        except Exception as exc:
            logger.error("Ollama generate_json failed: %s", exc)
            raise

    def name(self) -> str:
        return f"ollama/{self.model}"

    async def is_available(self) -> bool:
        try:
            client = self._get_client()
            await client.list()
            return True
        except Exception:
            return False
