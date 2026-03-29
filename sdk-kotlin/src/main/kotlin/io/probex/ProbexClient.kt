package io.probex

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import io.probex.models.*
import kotlinx.serialization.json.Json

/**
 * Kotlin coroutine-based client for the PROBEX REST API.
 *
 * ```kotlin
 * val client = ProbexClient("http://localhost:9712")
 * val profile = client.scan(ScanRequest("https://api.example.com"))
 * val results = client.run()
 * println("Passed: ${results.passed}")
 * client.close()
 * ```
 */
class ProbexClient(
    baseUrl: String = "http://localhost:9712",
    private val httpClient: HttpClient = defaultClient(),
) : AutoCloseable {

    private val baseUrl: String = baseUrl.trimEnd('/')

    companion object {
        private fun defaultClient(): HttpClient = HttpClient(CIO) {
            install(ContentNegotiation) {
                json(Json {
                    ignoreUnknownKeys = true
                    isLenient = true
                })
            }
            engine {
                requestTimeout = 120_000
            }
        }
    }

    /** Check server health. */
    suspend fun health(): HealthResponse = doGet("/api/v1/health")

    /** Get the current API profile. */
    suspend fun getProfile(): ScanResult = doGet("/api/v1/profile")

    /** Scan an API target. */
    suspend fun scan(request: ScanRequest): ScanResult = doPost("/api/v1/scan", request)

    /** Run tests with options. */
    suspend fun run(request: RunRequest = RunRequest()): TestResult =
        doPost("/api/v1/run", request)

    /** Get the latest test results. */
    suspend fun getResults(): TestResult = doGet("/api/v1/results")

    // --- AI endpoints ---

    /** Check AI brain health. */
    suspend fun aiHealth(): AIHealthResponse = doGet("/api/v1/ai/health")

    /** Generate AI-powered test scenarios. */
    suspend fun aiScenarios(request: ScenarioRequest): ScenarioResponse =
        doPost("/api/v1/ai/scenarios", request)

    /** Generate tests from natural language description. */
    suspend fun aiNLToTest(request: NLTestRequest): NLTestResponse =
        doPost("/api/v1/ai/nl-to-test", request)

    private suspend inline fun <reified T> doGet(path: String): T {
        val response = httpClient.get("$baseUrl$path")
        if (!response.status.isSuccess()) {
            val body = response.bodyAsText()
            throw ProbexException("GET $path returned ${response.status.value}: $body")
        }
        return response.body()
    }

    private suspend inline fun <reified R, reified T> doPost(path: String, body: R): T {
        val response = httpClient.post("$baseUrl$path") {
            contentType(ContentType.Application.Json)
            setBody(body)
        }
        if (!response.status.isSuccess()) {
            val text = response.bodyAsText()
            throw ProbexException("POST $path returned ${response.status.value}: $text")
        }
        return response.body()
    }

    override fun close() {
        httpClient.close()
    }
}
