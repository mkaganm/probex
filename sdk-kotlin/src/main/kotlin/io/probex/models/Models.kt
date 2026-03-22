package io.probex.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class HealthResponse(
    val status: String,
    val version: String,
)

@Serializable
data class Endpoint(
    val id: String? = null,
    val method: String,
    val path: String,
    @SerialName("base_url") val baseUrl: String? = null,
    val tags: List<String> = emptyList(),
)

@Serializable
data class ScanRequest(
    @SerialName("base_url") val baseUrl: String,
    @SerialName("max_depth") val maxDepth: Int = 3,
    val concurrency: Int = 10,
)

@Serializable
data class ScanResult(
    val id: String? = null,
    val name: String? = null,
    @SerialName("base_url") val baseUrl: String? = null,
    val endpoints: List<Endpoint> = emptyList(),
) {
    val endpointCount: Int get() = endpoints.size
}

@Serializable
data class RunRequest(
    val categories: List<String>? = null,
    val concurrency: Int? = null,
    val timeout: Int? = null,
)

@Serializable
data class SingleTestResult(
    @SerialName("test_case_id") val testCaseId: String? = null,
    @SerialName("test_name") val testName: String? = null,
    val status: String = "",
    val category: String? = null,
    val severity: String? = null,
    val duration: Long = 0,
    val error: String? = null,
) {
    val isPassed: Boolean get() = status == "passed"
    val isFailed: Boolean get() = status == "failed"
}

@Serializable
data class TestResult(
    @SerialName("profile_id") val profileId: String? = null,
    @SerialName("total_tests") val totalTests: Int = 0,
    val passed: Int = 0,
    val failed: Int = 0,
    val errors: Int = 0,
    val skipped: Int = 0,
    val duration: Long = 0,
    @SerialName("started_at") val startedAt: String? = null,
    @SerialName("finished_at") val finishedAt: String? = null,
    @SerialName("by_severity") val bySeverity: Map<String, Int>? = null,
    @SerialName("by_category") val byCategory: Map<String, Int>? = null,
    val results: List<SingleTestResult> = emptyList(),
) {
    val isSuccess: Boolean get() = failed == 0 && errors == 0

    fun failuresAtSeverity(vararg severities: String): Int =
        severities.sumOf { bySeverity?.getOrDefault(it, 0) ?: 0 }
}
