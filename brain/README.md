# PROBEX Brain — AI Analysis Service

Python FastAPI service providing AI-powered test generation, security analysis, and anomaly classification for PROBEX.

## Quick Start

```bash
cd brain

# Install with dev dependencies
python -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"

# Run the brain service
probex-brain --port 9711
# or
python -m probex_brain.server --port 9711
```

## Architecture

```
probex-brain (FastAPI, port 9711)
├── ai/
│   ├── router.py        — AI provider routing (local/cloud/hybrid/offline)
│   ├── local_provider.py — Ollama integration
│   ├── cloud_provider.py — Anthropic Claude integration
│   └── prompts.py       — System prompts for each task
├── generator/
│   ├── scenario.py      — Test scenario generation
│   ├── security.py      — OWASP security analysis
│   └── nl_to_test.py    — Natural language to test conversion
├── analysis/
│   └── anomaly.py       — Anomaly classification
└── models/
    ├── config.py         — AIConfig, ServerConfig
    └── schemas.py        — Request/response Pydantic models
```

## AI Modes

Configured via `PROBEX_AI_MODE` environment variable:

| Mode | Description |
|------|-------------|
| `offline` | No AI provider — returns errors for AI requests |
| `local` | Ollama (default model: `qwen3:4b`) — fully private, no data leaves your machine |
| `cloud` | Anthropic Claude (`claude-sonnet-4-6`) — higher quality, requires API key |
| `hybrid` | Tries local first, falls back to cloud on failure |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (returns status, version, ai_mode, model) |
| POST | `/api/v1/scenarios/generate` | Generate test scenarios from endpoint info |
| POST | `/api/v1/security/analyze` | Security analysis with findings and remediation |
| POST | `/api/v1/nl-to-test` | Convert natural language description to test cases |
| POST | `/api/v1/anomaly/classify` | Classify observed anomaly (degradation, spike, etc.) |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROBEX_AI_MODE` | `offline` | AI mode: `offline`, `local`, `cloud`, `hybrid` |
| `PROBEX_LOCAL_MODEL` | `qwen3:4b` | Ollama model name |
| `PROBEX_CLOUD_API_KEY` | — | Anthropic API key (required for cloud/hybrid mode) |
| `PROBEX_CLOUD_MODEL` | `claude-sonnet-4-6` | Claude model ID |
| `PROBEX_MAX_MONTHLY_COST` | `20.0` | Cloud cost limit (USD) |
| `PROBEX_BRAIN_HOST` | `127.0.0.1` | Bind address |
| `PROBEX_BRAIN_PORT` | `9711` | Listen port |

## Integration with Go Server

The brain runs as a subprocess managed by the Go CLI:

```bash
# Managed subprocess (Go starts the brain automatically)
probex serve --ai

# Custom port
probex serve --ai --ai-port 9711

# Connect to an external brain instance
probex serve --ai-url http://brain-host:9711
```

The Go server proxies SDK requests at `/api/v1/ai/*` to the brain's endpoints.

## Testing

```bash
source .venv/bin/activate
pytest tests/ -v
```

## Dependencies

- Python >= 3.11
- FastAPI + Uvicorn
- Ollama Python SDK (local AI)
- Anthropic Python SDK (cloud AI)
- Pydantic v2 (schema validation)
