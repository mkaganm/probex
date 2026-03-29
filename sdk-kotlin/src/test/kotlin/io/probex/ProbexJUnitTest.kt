package io.probex

import io.probex.models.SingleTestResult
import io.probex.models.TestResult
import kotlin.test.Test
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class ProbexJUnitTest {

    private fun passingResult() = TestResult(
        profileId = "prof-ok",
        totalTests = 3,
        passed = 3,
        failed = 0,
        errors = 0,
        results = listOf(
            SingleTestResult(testCaseId = "t1", testName = "Auth", status = "passed", severity = "high"),
            SingleTestResult(testCaseId = "t2", testName = "Injection", status = "passed", severity = "critical"),
            SingleTestResult(testCaseId = "t3", testName = "Rate limit", status = "passed", severity = "low"),
        ),
        bySeverity = mapOf("critical" to 0, "high" to 0, "low" to 0),
    )

    private fun failingResult() = TestResult(
        profileId = "prof-fail",
        totalTests = 3,
        passed = 1,
        failed = 2,
        errors = 0,
        results = listOf(
            SingleTestResult(testCaseId = "t1", testName = "Auth", status = "passed", severity = "high"),
            SingleTestResult(testCaseId = "t2", testName = "IDOR", status = "failed", severity = "high", error = "Unauthorized access"),
            SingleTestResult(testCaseId = "t3", testName = "SQL injection", status = "failed", severity = "critical", error = "Payload reflected"),
        ),
        bySeverity = mapOf("critical" to 1, "high" to 1),
    )

    @Test
    fun `assertAllPass succeeds for passing result`() {
        ProbexJUnit.assertAllPass(passingResult())
    }

    @Test
    fun `assertAllPass throws AssertionError for failing result`() {
        val ex = assertFailsWith<AssertionError> {
            ProbexJUnit.assertAllPass(failingResult())
        }
        assertTrue(ex.message!!.contains("2 test(s) failed"))
        assertTrue(ex.message!!.contains("IDOR"))
        assertTrue(ex.message!!.contains("SQL injection"))
    }

    @Test
    fun `assertNoHighSeverity succeeds when no high or critical failures`() {
        ProbexJUnit.assertNoHighSeverity(passingResult())
    }

    @Test
    fun `assertNoHighSeverity throws for critical failures`() {
        val ex = assertFailsWith<AssertionError> {
            ProbexJUnit.assertNoHighSeverity(failingResult())
        }
        assertTrue(ex.message!!.contains("critical/high severity failures"))
    }

    @Test
    fun `assertAllPass succeeds for zero-test result`() {
        val empty = TestResult(totalTests = 0, passed = 0, failed = 0, errors = 0)
        ProbexJUnit.assertAllPass(empty)
    }

    @Test
    fun `assertNoHighSeverity succeeds when bySeverity is null`() {
        val result = TestResult(totalTests = 1, passed = 1, failed = 0, errors = 0, bySeverity = null)
        ProbexJUnit.assertNoHighSeverity(result)
    }

    @Test
    fun `assertAllPass throws when errors present even if no failures`() {
        val result = TestResult(totalTests = 2, passed = 1, failed = 0, errors = 1)
        val ex = assertFailsWith<AssertionError> {
            ProbexJUnit.assertAllPass(result)
        }
        assertTrue(ex.message!!.contains("1 error(s)"))
    }
}
