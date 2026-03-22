"""Configuration models for the PROBEX AI brain service."""

from __future__ import annotations

from pydantic import BaseModel


class AIConfig(BaseModel):
    mode: str = "offline"  # local, cloud, hybrid, offline
    local_provider: str = "ollama"
    local_model: str = "qwen3:4b"
    cloud_provider: str = "anthropic"
    cloud_model: str = "claude-sonnet-4-6"
    cloud_api_key: str = ""
    max_monthly_cost: float = 20.0
    prefer_local: bool = True


class ServerConfig(BaseModel):
    host: str = "127.0.0.1"
    port: int = 9711
    log_level: str = "info"
