package io.probex

import io.probex.models.RunRequest
import io.probex.models.TestResult
import kotlinx.coroutines.runBlocking

/**
 * JUnit 5 helper for running PROBEX tests within standard Kotlin test suites.
 *
 * ```kotlin
 * class ApiTest {
 *     @Test
 *     fun `all endpoints should pass probex tests`() {
 *         val result = ProbexJUnit.scanAndRun("https://api.example.com")
 *         assert(result.isSuccess) { "PROBEX: ${result.failed} failures" }
 *     }
 * }
 * ```
 */
object ProbexJUnit {

    /** Scan a target and run all tests, blocking the current thread. */
    fun scanAndRun(
        targetUrl: String,
        serverUrl: String = "http://localhost:9712",
        categories: List<String>? = null,
        maxDepth: Int = 3,
    ): TestResult = runBlocking {
        ProbexClient(serverUrl).use { client ->
            client.scan(io.probex.models.ScanRequest(targetUrl, maxDepth))
            client.run(RunRequest(categories))
        }
    }

    /** Run tests against an already-scanned profile. */
    fun run(
        serverUrl: String = "http://localhost:9712",
        categories: List<String>? = null,
    ): TestResult = runBlocking {
        ProbexClient(serverUrl).use { client ->
            client.run(RunRequest(categories))
        }
    }

    /** Assert all tests pass, throwing AssertionError on failure. */
    fun assertAllPass(result: TestResult) {
        if (!result.isSuccess) {
            val failures = result.results.filter { it.isFailed }
            val summary = failures.joinToString("\n") { "  - ${it.testName}: ${it.error ?: it.status}" }
            throw AssertionError(
                "PROBEX: ${result.failed} test(s) failed, ${result.errors} error(s)\n$summary"
            )
        }
    }

    /** Assert no critical/high severity failures. */
    fun assertNoHighSeverity(result: TestResult) {
        val count = result.failuresAtSeverity("critical", "high")
        if (count > 0) {
            throw AssertionError("PROBEX: $count critical/high severity failures detected")
        }
    }
}
