# JavaScript/TypeScript SDK

The PROBEX JS SDK (`@probex/sdk`) provides a TypeScript client for integrating PROBEX into Node.js projects.

## Installation

```bash
npm install @probex/sdk
```

## Prerequisites

The PROBEX server must be running:

```bash
probex serve
# Server starts at http://localhost:9712
```

## Usage

### Basic Client

```typescript
import { ProbexClient } from '@probex/sdk';

const client = new ProbexClient('http://localhost:9712');

// Health check
const health = await client.health();
console.log(health.version);

// Scan an API
const profile = await client.scan('https://api.example.com');
console.log(`Found ${profile.endpoints.length} endpoints`);

// Run tests
const results = await client.run();
console.log(`${results.passed} passed, ${results.failed} failed`);

// Get results
const latest = await client.getResults();
```

### Jest Integration

```typescript
import { Probex } from '@probex/jest';

describe('API Tests', () => {
  it('should pass all probex tests', async () => {
    const results = await Probex.run({
      baseUrl: 'http://localhost:3000',
    });
    expect(results.failed).toBe(0);
  });

  it('should have no critical security findings', async () => {
    const results = await Probex.run({
      categories: ['security'],
    });
    expect(results.failuresAtSeverity('critical')).toBe(0);
  });
});
```

### Vitest Integration

```typescript
import { Probex } from '@probex/vitest';
import { expect, test } from 'vitest';

test('API security', async () => {
  const results = await Probex.run({ categories: ['security'] });
  expect(results.isSuccess).toBe(true);
});
```

## GitHub Actions

```yaml
name: API Tests
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: probex/action@v1
        with:
          target-url: https://staging-api.example.com
          fail-on: critical,high
```

### Action Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `target-url` | API URL to test | (required) |
| `server-url` | PROBEX server URL | `http://localhost:9712` |
| `fail-on` | Severity levels that cause failure | `critical,high` |
| `categories` | Test categories to run | (all) |
| `max-depth` | Scan depth | `3` |

## API Reference

### `ProbexClient`

| Method | Returns | Description |
|--------|---------|-------------|
| `health()` | `HealthResponse` | Server health check |
| `scan(url, maxDepth?)` | `ScanResult` | Scan an API |
| `run(categories?)` | `TestResult` | Run tests |
| `getProfile()` | `ScanResult` | Get current profile |
| `getResults()` | `TestResult` | Get latest results |
