package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;

/** Result of a single test case execution. */
public class SingleTestResult {
    @JsonProperty("test_case_id")
    private String testCaseId;

    @JsonProperty("test_name")
    private String testName;

    private String status; // passed, failed, error, skipped
    private String category;
    private String severity;
    private long duration;
    private String error;

    public SingleTestResult() {}

    public String getTestCaseId() { return testCaseId; }
    public void setTestCaseId(String testCaseId) { this.testCaseId = testCaseId; }
    public String getTestName() { return testName; }
    public void setTestName(String testName) { this.testName = testName; }
    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }
    public String getCategory() { return category; }
    public void setCategory(String category) { this.category = category; }
    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }
    public long getDuration() { return duration; }
    public void setDuration(long duration) { this.duration = duration; }
    public String getError() { return error; }
    public void setError(String error) { this.error = error; }

    public boolean isPassed() { return "passed".equals(status); }
    public boolean isFailed() { return "failed".equals(status); }
}
