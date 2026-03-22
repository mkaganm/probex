"""Abstract base class for AI providers."""

from __future__ import annotations

from abc import ABC, abstractmethod


class AIProvider(ABC):
    """Base interface for all AI providers (local and cloud)."""

    @abstractmethod
    async def generate(self, prompt: str, system: str = "", max_tokens: int = 4096) -> str:
        """Generate a text completion from the given prompt."""
        ...

    @abstractmethod
    async def generate_json(
        self, prompt: str, system: str = "", max_tokens: int = 4096
    ) -> dict:
        """Generate a JSON response from the given prompt."""
        ...

    @abstractmethod
    def name(self) -> str:
        """Return the provider/model name."""
        ...

    @abstractmethod
    async def is_available(self) -> bool:
        """Check whether this provider is currently reachable."""
        ...
