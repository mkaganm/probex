# Kotlin SDK

The PROBEX Kotlin SDK (`io.probex:probex-sdk-kotlin`) provides a coroutine-based client with an idiomatic Kotlin DSL.

## Installation

### Gradle (Kotlin DSL)

```kotlin
implementation("io.probex:probex-sdk-kotlin:1.0.0")
```

## Prerequisites

The PROBEX server must be running:

```bash
probex serve
```

## Usage

### Basic Client

```kotlin
val client = ProbexClient("http://localhost:9712")

// Scan
val profile = client.scan(ScanRequest("https://api.example.com"))
println("Found ${profile.endpointCount} endpoints")

// Run tests
val results = client.run()
println("${results.passed} passed, ${results.failed} failed")

// Cleanup
client.close()
```

### DSL

```kotlin
val result = probex("http://localhost:9712") {
    scan("https://api.example.com")
    run(categories = listOf("security"))
}

assert(result.isSuccess) { "PROBEX: ${result.failed} failures" }
```

### JUnit 5 Helper

```kotlin
class ApiTest {
    @Test
    fun `all endpoints should pass`() {
        val result = ProbexJUnit.scanAndRun("https://api.example.com")
        ProbexJUnit.assertAllPass(result)
    }

    @Test
    fun `no critical security findings`() {
        val result = ProbexJUnit.run(categories = listOf("security"))
        ProbexJUnit.assertNoHighSeverity(result)
    }
}
```

### Coroutine Support

All client methods are `suspend` functions and work natively with Kotlin coroutines:

```kotlin
runBlocking {
    val client = ProbexClient()
    val health = client.health()
    println("Server: ${health.version}")

    val results = client.run()
    results.results
        .filter { it.isFailed }
        .forEach { println("FAILED: ${it.testName}") }

    client.close()
}
```

## API Reference

### `ProbexClient`

| Method | Returns | Description |
|--------|---------|-------------|
| `suspend health()` | `HealthResponse` | Server health check |
| `suspend scan(ScanRequest)` | `ScanResult` | Scan an API |
| `suspend run(RunRequest)` | `TestResult` | Run tests |
| `suspend getProfile()` | `ScanResult` | Get current profile |
| `suspend getResults()` | `TestResult` | Get latest results |
| `close()` | — | Close HTTP client |

### `TestResult`

| Property | Type | Description |
|----------|------|-------------|
| `isSuccess` | `Boolean` | True if no failures or errors |
| `totalTests` | `Int` | Total test count |
| `passed` | `Int` | Passed tests |
| `failed` | `Int` | Failed tests |
| `results` | `List<SingleTestResult>` | Individual results |

### `ProbexJUnit`

| Method | Description |
|--------|-------------|
| `scanAndRun(url, ...)` | Blocking scan + run |
| `run(...)` | Blocking run only |
| `assertAllPass(result)` | Throws if any test failed |
| `assertNoHighSeverity(result)` | Throws if critical/high failures exist |

## Dependencies

- Kotlin 2.1+
- Ktor Client (CIO engine)
- kotlinx.serialization
- kotlinx.coroutines
