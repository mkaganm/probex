package io.probex

import io.probex.models.RunRequest
import io.probex.models.ScanRequest
import io.probex.models.TestResult

/**
 * DSL for concise PROBEX usage in Kotlin tests:
 *
 * ```kotlin
 * val result = probex("http://localhost:9712") {
 *     scan("https://api.example.com")
 *     run()
 * }
 * assert(result.isSuccess)
 * ```
 */
class ProbexDslBuilder(private val client: ProbexClient) {
    private var lastResult: TestResult? = null

    suspend fun scan(targetUrl: String, maxDepth: Int = 3, concurrency: Int = 10) {
        client.scan(ScanRequest(targetUrl, maxDepth, concurrency))
    }

    suspend fun run(
        categories: List<String>? = null,
        concurrency: Int? = null,
        timeout: Int? = null,
    ): TestResult {
        val result = client.run(RunRequest(categories, concurrency, timeout))
        lastResult = result
        return result
    }

    fun result(): TestResult = lastResult ?: throw ProbexException("No test run executed yet")
}

suspend fun probex(
    serverUrl: String = "http://localhost:9712",
    block: suspend ProbexDslBuilder.() -> Unit,
): TestResult {
    val client = ProbexClient(serverUrl)
    return client.use {
        val builder = ProbexDslBuilder(it)
        builder.block()
        builder.result()
    }
}
