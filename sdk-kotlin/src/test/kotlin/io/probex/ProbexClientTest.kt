package io.probex

import io.ktor.client.*
import io.ktor.client.engine.mock.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import io.probex.models.RunRequest
import io.probex.models.ScanRequest
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class ProbexClientTest {

    private val healthJson = """{"status":"ok","version":"1.2.3"}"""

    private val scanResultJson = """{
        "id": "scan-001",
        "name": "Test API",
        "base_url": "https://api.example.com",
        "endpoints": [
            {"method": "GET", "path": "/users", "tags": ["users"]},
            {"method": "POST", "path": "/users", "tags": ["users"]}
        ]
    }"""

    private val testResultJson = """{
        "profile_id": "prof-001",
        "total_tests": 5,
        "passed": 4,
        "failed": 1,
        "errors": 0,
        "skipped": 0,
        "duration": 1200,
        "by_severity": {"high": 1, "low": 0},
        "results": [
            {"test_case_id": "tc-1", "test_name": "Auth check", "status": "passed", "category": "auth", "severity": "high", "duration": 100},
            {"test_case_id": "tc-2", "test_name": "SQL injection", "status": "passed", "category": "injection", "severity": "critical", "duration": 200},
            {"test_case_id": "tc-3", "test_name": "XSS check", "status": "passed", "category": "injection", "severity": "medium", "duration": 150},
            {"test_case_id": "tc-4", "test_name": "Rate limit", "status": "passed", "category": "rate-limit", "severity": "low", "duration": 300},
            {"test_case_id": "tc-5", "test_name": "IDOR", "status": "failed", "category": "auth", "severity": "high", "duration": 450, "error": "Unauthorized access"}
        ]
    }"""

    private fun buildMockClient(
        customHandler: (MockRequestHandleScope.(HttpRequestData) -> HttpResponseData)? = null,
    ): HttpClient = HttpClient(MockEngine { request ->
        if (customHandler != null) {
            customHandler(request)
        } else {
            val jsonHeaders = headersOf(HttpHeaders.ContentType, "application/json")
            when (request.url.encodedPath) {
                "/api/v1/health" -> respond(healthJson, HttpStatusCode.OK, jsonHeaders)
                "/api/v1/profile" -> respond(scanResultJson, HttpStatusCode.OK, jsonHeaders)
                "/api/v1/scan" -> respond(scanResultJson, HttpStatusCode.OK, jsonHeaders)
                "/api/v1/run" -> respond(testResultJson, HttpStatusCode.OK, jsonHeaders)
                "/api/v1/results" -> respond(testResultJson, HttpStatusCode.OK, jsonHeaders)
                else -> respond("Not found", HttpStatusCode.NotFound)
            }
        }
    }) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true; isLenient = true })
        }
    }

    private fun buildClient(
        customHandler: (MockRequestHandleScope.(HttpRequestData) -> HttpResponseData)? = null,
    ): ProbexClient = ProbexClient(
        baseUrl = "http://localhost:9712",
        httpClient = buildMockClient(customHandler),
    )

    @Test
    fun `health returns parsed HealthResponse`() = runTest {
        val client = buildClient()
        val result = client.health()
        assertEquals("ok", result.status)
        assertEquals("1.2.3", result.version)
    }

    @Test
    fun `getProfile returns parsed ScanResult`() = runTest {
        val client = buildClient()
        val result = client.getProfile()
        assertEquals("scan-001", result.id)
        assertEquals("Test API", result.name)
        assertEquals("https://api.example.com", result.baseUrl)
        assertEquals(2, result.endpoints.size)
        assertEquals("GET", result.endpoints[0].method)
        assertEquals("/users", result.endpoints[0].path)
    }

    @Test
    fun `scan sends POST and returns ScanResult`() = runTest {
        val client = buildClient()
        val result = client.scan(ScanRequest("https://api.example.com", maxDepth = 5, concurrency = 20))
        assertEquals("scan-001", result.id)
        assertEquals(2, result.endpointCount)
    }

    @Test
    fun `run sends POST and returns TestResult`() = runTest {
        val client = buildClient()
        val result = client.run(RunRequest(categories = listOf("auth")))
        assertEquals(5, result.totalTests)
        assertEquals(4, result.passed)
        assertEquals(1, result.failed)
        assertEquals(0, result.errors)
        assertEquals(5, result.results.size)
    }

    @Test
    fun `run with default request works`() = runTest {
        val client = buildClient()
        val result = client.run()
        assertEquals(5, result.totalTests)
    }

    @Test
    fun `getResults returns parsed TestResult`() = runTest {
        val client = buildClient()
        val result = client.getResults()
        assertEquals("prof-001", result.profileId)
        assertEquals(1, result.failed)
        assertTrue(result.results.any { it.isFailed })
        assertTrue(result.results.any { it.isPassed })
    }

    @Test
    fun `GET error throws ProbexException with status and body`() = runTest {
        val client = buildClient { _ ->
            respond("server down", HttpStatusCode.InternalServerError)
        }
        val ex = assertFailsWith<ProbexException> {
            client.health()
        }
        assertTrue(ex.message!!.contains("GET /api/v1/health returned 500"))
        assertTrue(ex.message!!.contains("server down"))
    }

    @Test
    fun `POST error throws ProbexException with status and body`() = runTest {
        val client = buildClient { _ ->
            respond("bad request", HttpStatusCode.BadRequest)
        }
        val ex = assertFailsWith<ProbexException> {
            client.scan(ScanRequest("https://example.com"))
        }
        assertTrue(ex.message!!.contains("POST /api/v1/scan returned 400"))
        assertTrue(ex.message!!.contains("bad request"))
    }

    @Test
    fun `404 on getResults throws ProbexException`() = runTest {
        val client = buildClient { _ ->
            respond("not found", HttpStatusCode.NotFound)
        }
        val ex = assertFailsWith<ProbexException> {
            client.getResults()
        }
        assertTrue(ex.message!!.contains("404"))
    }

    @Test
    fun `close does not throw`() {
        val client = buildClient()
        client.close()
    }

    @Test
    fun `scan request path is correct`() = runTest {
        var capturedPath = ""
        val httpClient = HttpClient(MockEngine { request ->
            capturedPath = request.url.encodedPath
            val jsonHeaders = headersOf(HttpHeaders.ContentType, "application/json")
            respond(scanResultJson, HttpStatusCode.OK, jsonHeaders)
        }) {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true; isLenient = true })
            }
        }
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = httpClient)
        client.scan(ScanRequest("https://example.com"))
        assertEquals("/api/v1/scan", capturedPath)
    }

    @Test
    fun `run request path is correct`() = runTest {
        var capturedPath = ""
        val httpClient = HttpClient(MockEngine { request ->
            capturedPath = request.url.encodedPath
            val jsonHeaders = headersOf(HttpHeaders.ContentType, "application/json")
            respond(testResultJson, HttpStatusCode.OK, jsonHeaders)
        }) {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true; isLenient = true })
            }
        }
        val client = ProbexClient(baseUrl = "http://localhost:9712", httpClient = httpClient)
        client.run()
        assertEquals("/api/v1/run", capturedPath)
    }
}
