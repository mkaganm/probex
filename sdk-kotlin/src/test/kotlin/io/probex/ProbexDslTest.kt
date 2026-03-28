package io.probex

import io.ktor.client.*
import io.ktor.client.engine.mock.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ProbexDslTest {

    private val scanResultJson = """{
        "id": "scan-dsl",
        "name": "DSL API",
        "base_url": "https://api.example.com",
        "endpoints": [{"method": "GET", "path": "/health", "tags": []}]
    }"""

    private val testResultJson = """{
        "profile_id": "prof-dsl",
        "total_tests": 3,
        "passed": 3,
        "failed": 0,
        "errors": 0,
        "skipped": 0,
        "duration": 500,
        "results": [
            {"test_case_id": "tc-1", "test_name": "Check A", "status": "passed", "severity": "low", "duration": 100},
            {"test_case_id": "tc-2", "test_name": "Check B", "status": "passed", "severity": "medium", "duration": 200},
            {"test_case_id": "tc-3", "test_name": "Check C", "status": "passed", "severity": "high", "duration": 200}
        ]
    }"""

    private fun buildMockClient(): HttpClient = HttpClient(MockEngine { request ->
        val jsonHeaders = headersOf(HttpHeaders.ContentType, "application/json")
        when (request.url.encodedPath) {
            "/api/v1/scan" -> respond(scanResultJson, HttpStatusCode.OK, jsonHeaders)
            "/api/v1/run" -> respond(testResultJson, HttpStatusCode.OK, jsonHeaders)
            else -> respond("not found", HttpStatusCode.NotFound)
        }
    }) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true; isLenient = true })
        }
    }

    @Test
    fun `DSL scan and run returns TestResult`() = runTest {
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = buildMockClient())
        val builder = ProbexDslBuilder(client)
        builder.scan("https://api.example.com")
        val result = builder.run()
        assertEquals(3, result.totalTests)
        assertEquals(3, result.passed)
        assertEquals(0, result.failed)
        assertTrue(result.isSuccess)
    }

    @Test
    fun `DSL result returns last run result`() = runTest {
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = buildMockClient())
        val builder = ProbexDslBuilder(client)
        builder.scan("https://api.example.com")
        builder.run()
        val result = builder.result()
        assertEquals("prof-dsl", result.profileId)
        assertEquals(3, result.totalTests)
    }

    @Test
    fun `DSL result before run throws ProbexException`() {
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = buildMockClient())
        val builder = ProbexDslBuilder(client)
        val ex = assertFailsWith<ProbexException> {
            builder.result()
        }
        assertEquals("No test run executed yet", ex.message)
    }

    @Test
    fun `DSL run with parameters works`() = runTest {
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = buildMockClient())
        val builder = ProbexDslBuilder(client)
        builder.scan("https://api.example.com", maxDepth = 5, concurrency = 20)
        val result = builder.run(categories = listOf("auth"), concurrency = 4, timeout = 30)
        assertFalse(result.results.isEmpty())
        assertEquals(3, result.results.size)
    }

    @Test
    fun `DSL multiple runs keeps latest result`() = runTest {
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = buildMockClient())
        val builder = ProbexDslBuilder(client)
        builder.scan("https://api.example.com")
        builder.run()
        val firstResult = builder.result()
        builder.run(categories = listOf("injection"))
        val secondResult = builder.result()
        // Both come from the same mock, so they should be equal in content
        assertEquals(firstResult.totalTests, secondResult.totalTests)
    }
}
