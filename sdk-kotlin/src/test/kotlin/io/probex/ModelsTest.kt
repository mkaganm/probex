package io.probex

import io.probex.models.*
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ModelsTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `TestResult isSuccess when no failures`() {
        val result = TestResult(totalTests = 5, passed = 5, failed = 0, errors = 0)
        assertTrue(result.isSuccess)
    }

    @Test
    fun `TestResult not success when failures exist`() {
        val result = TestResult(totalTests = 5, passed = 3, failed = 2, errors = 0)
        assertFalse(result.isSuccess)
    }

    @Test
    fun `TestResult failuresAtSeverity counts correctly`() {
        val result = TestResult(
            totalTests = 10,
            passed = 6,
            failed = 4,
            bySeverity = mapOf("critical" to 1, "high" to 2, "medium" to 1),
        )
        assertEquals(3, result.failuresAtSeverity("critical", "high"))
        assertEquals(1, result.failuresAtSeverity("medium"))
        assertEquals(0, result.failuresAtSeverity("low"))
    }

    @Test
    fun `SingleTestResult status helpers`() {
        val passed = SingleTestResult(status = "passed")
        val failed = SingleTestResult(status = "failed")
        assertTrue(passed.isPassed)
        assertFalse(passed.isFailed)
        assertTrue(failed.isFailed)
        assertFalse(failed.isPassed)
    }

    @Test
    fun `ScanResult endpointCount`() {
        val result = ScanResult(
            endpoints = listOf(
                Endpoint(method = "GET", path = "/users"),
                Endpoint(method = "POST", path = "/users"),
            ),
        )
        assertEquals(2, result.endpointCount)
    }

    @Test
    fun `deserialize health response from JSON`() {
        val raw = """{"status":"ok","version":"1.0.0"}"""
        val health = json.decodeFromString<HealthResponse>(raw)
        assertEquals("ok", health.status)
        assertEquals("1.0.0", health.version)
    }

    @Test
    fun `deserialize test result with unknown fields`() {
        val raw = """{"total_tests":3,"passed":2,"failed":1,"errors":0,"extra_field":"ignored"}"""
        val result = json.decodeFromString<TestResult>(raw)
        assertEquals(3, result.totalTests)
        assertEquals(1, result.failed)
    }
}
