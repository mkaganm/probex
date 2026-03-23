# Plugins

PROBEX supports a plugin system for extending test generation, reporting, and lifecycle hooks.

## Plugin Types

| Type | Interface | Purpose |
|------|-----------|---------|
| Generator | `GeneratorPlugin` | Produce additional test cases |
| Reporter | `ReporterPlugin` | Custom report formats |
| Hook | `HookPlugin` | Lifecycle events (before/after scan/run) |

## Go Interface Plugins

Register plugins programmatically when embedding PROBEX as a library:

```go
package main

import "github.com/mkaganm/probex/internal/plugin"

type MyGenerator struct{}

func (g *MyGenerator) Name() string        { return "my-generator" }
func (g *MyGenerator) Version() string     { return "1.0.0" }
func (g *MyGenerator) Description() string { return "Custom test generator" }

func (g *MyGenerator) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
    // Generate custom test cases
    return tests, nil
}

func main() {
    registry := plugin.NewRegistry()
    registry.RegisterGenerator(&MyGenerator{})
}
```

## External HTTP Plugins

External plugins communicate via HTTP JSON-RPC. They can be written in any language.

### Plugin Discovery

PROBEX calls `GET /meta` on the plugin's HTTP server to discover capabilities:

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "Custom plugin",
  "capabilities": ["generator", "hook"]
}
```

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/meta` | Plugin metadata and capabilities |
| POST | `/generate` | Generate test cases (generator plugins) |
| POST | `/report` | Process results (reporter plugins) |
| POST | `/hooks/before-scan` | Before scan hook |
| POST | `/hooks/after-scan` | After scan hook |
| POST | `/hooks/before-run` | Before run hook |
| POST | `/hooks/after-run` | After run hook |

### Generator Endpoint

**Request:**
```json
{
  "endpoint": {
    "method": "GET",
    "path": "/users",
    "base_url": "https://api.example.com"
  }
}
```

**Response:**
```json
{
  "test_cases": [
    {
      "name": "custom-test-1",
      "category": "custom",
      "severity": "medium",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/users",
        "headers": {}
      },
      "assertions": [
        {"type": "status_code", "operator": "eq", "expected": 200}
      ]
    }
  ]
}
```

### Example: Python Plugin

```python
from fastapi import FastAPI

app = FastAPI()

@app.get("/meta")
def meta():
    return {
        "name": "python-security-plugin",
        "version": "1.0.0",
        "capabilities": ["generator"]
    }

@app.post("/generate")
def generate(request: dict):
    endpoint = request["endpoint"]
    # Custom test generation logic
    return {"test_cases": [...]}
```

### Hook Lifecycle

```
probex scan:
  1. RunBeforeScan(profile)    → hooks/before-scan
  2. (scan executes)
  3. RunAfterScan(profile)     → hooks/after-scan

probex run:
  1. RunBeforeRun(tests)       → hooks/before-run
  2. (tests execute)
  3. RunAfterRun(results)      → hooks/after-run
```
