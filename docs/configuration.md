# Configuration

PROBEX is configured via `probex.yaml` in the project root.

## Initialize Configuration

```bash
probex config init
```

This creates `probex.yaml` with sensible defaults.

## View Current Configuration

```bash
probex config show
```

## Full Reference

```yaml
# PROBEX Configuration
version: "1"

# Target API
target:
  base_url: "https://api.example.com"
  auth_header: "Authorization: Bearer <token>"
  headers:
    X-Custom-Header: "value"

# Scan settings
scan:
  max_depth: 3              # Crawl depth limit
  concurrency: 10           # Parallel scan requests
  timeout: 30s              # Per-request timeout
  wordlist: ""              # Custom wordlist path (default: built-in)
  follow_links: true        # Follow links in responses

# Test run settings
run:
  concurrency: 5            # Parallel test executions
  timeout: 30s              # Per-test timeout
  categories: []            # Filter: [happy_path, edge_case, security, fuzz, ...]
  use_ai: false             # Enable AI-powered generation
  stop_on_fail: false       # Stop on first failure

# Watch mode settings
watch:
  interval: 5m              # Polling interval
  endpoints: []             # Specific endpoints to watch (empty = all)
  notify_slack: ""          # Slack webhook URL
  notify_webhook: ""        # Generic webhook URL

# CI/CD guard settings
guard:
  fail_on:                  # Severity levels that cause non-zero exit
    - critical
    - high
  report_file: ""           # Path for JUnit XML output

# AI settings
ai:
  mode: "offline"           # offline, local, cloud, hybrid
  local:
    provider: "ollama"      # ollama, llamacpp
    model: "qwen3:4b"
  cloud:
    provider: "anthropic"   # anthropic, openai
    model: "claude-sonnet-4-6"
    api_key: ""             # Or set ANTHROPIC_API_KEY env var
    use_for:                # Tasks that use cloud AI
      - security_analysis
      - complex_scenarios
    never_send: []          # Fields to strip before sending to cloud
  budget:
    max_monthly_cost: 20    # USD spending cap
    prefer_local_when_possible: true

# Report settings
report:
  format: "json"            # json, junit, html
  output: "stdout"          # stdout or file path
```

## AI Modes

| Mode | Description |
|------|-------------|
| `offline` | No AI features. Pure rule-based generation. |
| `local` | Use Ollama or similar local LLM. Private, no data leaves your machine. |
| `cloud` | Use Anthropic Claude or OpenAI. More capable but sends data to cloud. |
| `hybrid` | Local for most tasks, cloud for complex analysis. Budget-controlled. |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | API key for Claude (cloud mode) |
| `OPENAI_API_KEY` | API key for OpenAI (cloud mode) |
| `PROBEX_CONFIG` | Path to config file (overrides default) |
| `PROBEX_SERVER_URL` | Server URL for SDK communication |

## CLI Flag Precedence

CLI flags override config file values. For example:

```bash
# Config says concurrency: 5, but this run uses 20
probex run --concurrency 20
```
