# Java SDK

The PROBEX Java SDK (`io.probex:probex-sdk`) provides a client for integrating PROBEX into Java and Kotlin projects, with JUnit 5 and Maven/Gradle plugin support.

## Installation

### Maven

```xml
<dependency>
    <groupId>io.probex</groupId>
    <artifactId>probex-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```kotlin
implementation("io.probex:probex-sdk:1.0.0")
```

## Prerequisites

The PROBEX server must be running:

```bash
probex serve
```

## Usage

### Basic Client

```java
var client = new ProbexClient("http://localhost:9712");

// Scan
var profile = client.scan(new ScanRequest("https://api.example.com"));
System.out.println("Found " + profile.getEndpointCount() + " endpoints");

// Run tests
var results = client.run();
System.out.println(results.getPassed() + " passed, " + results.getFailed() + " failed");

// Check success
if (results.isSuccess()) {
    System.out.println("All tests passed!");
}
```

### JUnit 5 Extension

```java
@ExtendWith(ProbexExtension.class)
@ProbexConfig(baseUrl = "http://localhost:8080")
class ApiTests {

    @ProbexTest
    void allEndpointsShouldPass(ProbexResult result) {
        assertThat(result.getFailed()).isZero();
    }

    @ProbexTest(categories = {"security"})
    void noSecurityIssues(ProbexResult result) {
        assertThat(result.failuresAtSeverity("critical", "high")).isZero();
    }
}
```

## Maven Plugin

### Configuration

```xml
<plugin>
    <groupId>io.probex</groupId>
    <artifactId>probex-maven-plugin</artifactId>
    <version>1.0.0</version>
    <configuration>
        <targetUrl>http://localhost:8080</targetUrl>
        <serverUrl>http://localhost:9712</serverUrl>
        <failOn>critical,high</failOn>
    </configuration>
</plugin>
```

### Goals

```bash
mvn probex:scan    # Scan the target API
mvn probex:test    # Run tests (includes scan)
mvn probex:report  # Generate report
```

## Gradle Plugin

### Configuration

```kotlin
plugins {
    id("io.probex.gradle") version "1.0.0"
}

probex {
    targetUrl = "http://localhost:8080"
    serverUrl = "http://localhost:9712"
    failOn = listOf("critical", "high")
}
```

### Tasks

```bash
./gradlew probexScan    # Scan the target API
./gradlew probexTest    # Scan + run tests
./gradlew probexReport  # Generate report
```

## API Reference

### `ProbexClient`

| Method | Returns | Description |
|--------|---------|-------------|
| `health()` | `HealthResponse` | Server health check |
| `scan(ScanRequest)` | `ScanResult` | Scan an API |
| `run(RunRequest)` | `TestResult` | Run tests |
| `run()` | `TestResult` | Run tests with defaults |
| `getProfile()` | `ScanResult` | Get current profile |
| `getResults()` | `TestResult` | Get latest results |

### `TestResult`

| Method | Returns | Description |
|--------|---------|-------------|
| `isSuccess()` | `boolean` | True if no failures or errors |
| `failuresAtSeverity(String...)` | `int` | Count of failures at given severities |
| `getTotalTests()` | `int` | Total test count |
| `getPassed()` | `int` | Passed test count |
| `getFailed()` | `int` | Failed test count |
