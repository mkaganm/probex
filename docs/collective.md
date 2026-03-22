# Collective Intelligence

PROBEX includes an opt-in community pattern sharing system. Instances can anonymously share test patterns that found real issues, and pull community patterns to enrich local testing.

## Privacy

Only abstract patterns are shared. The following information is **never** transmitted:

- API URLs, hostnames, or IP addresses
- Authentication tokens or credentials
- Request/response bodies
- Headers or query parameters
- Any personally identifiable information

What **is** shared:
- Test category (e.g., "security", "edge_case")
- Severity level
- Abstract test type (e.g., "bola", "rate_limit")
- Anonymized description
- Effectiveness score (did the test find a real issue?)

## Usage

### Push Patterns

Share anonymized patterns from your latest test run:

```bash
probex collective push
# Anonymized 12 patterns from latest run
# Shared 12 patterns with the community
```

### Pull Patterns

Pull community patterns to use in your tests:

```bash
probex collective pull
# Pulled 45 community patterns (total available: 1203)

probex collective pull --min-score 0.9
# Only high-effectiveness patterns

probex collective pull --category security
# Only security patterns
```

### Use in Test Runs

```bash
probex run --collective
# Includes community patterns in test generation
```

## How It Works

### Pattern Extraction

After each test run, the **Anonymizer** processes results:

1. Groups results by category + severity
2. Strips all identifying information (URLs, IDs, tokens)
3. Computes an effectiveness score:
   - Failed tests (found real issues): score 0.8
   - Passed tests: score 0.5
4. Deduplicates by pattern signature

### Instance Identity

Each PROBEX installation generates a stable but non-identifying instance ID by hashing the hostname with a salt. This allows the hub to track pattern contributions without identifying users.

### Hub API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/collective/push` | Submit anonymized patterns |
| GET | `/api/v1/collective/pull` | Retrieve community patterns |

Query parameters for pull:
- `min_score` — Minimum effectiveness score (0.0-1.0)
- `category` — Filter by category (repeatable)

## Configuration

```bash
probex collective push --hub https://hub.probex.dev
probex collective pull --hub https://hub.probex.dev --min-score 0.8
```

For self-hosted hubs:

```bash
probex collective push --hub https://internal-hub.company.com
```
