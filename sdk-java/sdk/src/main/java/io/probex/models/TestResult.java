package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;
import java.util.Map;

/** Aggregate test run results, equivalent to Go's RunSummary. */
public class TestResult {
    @JsonProperty("profile_id")
    private String profileId;

    @JsonProperty("total_tests")
    private int totalTests;

    private int passed;
    private int failed;
    private int errors;
    private int skipped;
    private long duration;

    @JsonProperty("started_at")
    private String startedAt;

    @JsonProperty("finished_at")
    private String finishedAt;

    @JsonProperty("by_severity")
    private Map<String, Integer> bySeverity;

    @JsonProperty("by_category")
    private Map<String, Integer> byCategory;

    private List<SingleTestResult> results;

    public TestResult() {}

    public String getProfileId() { return profileId; }
    public void setProfileId(String profileId) { this.profileId = profileId; }
    public int getTotalTests() { return totalTests; }
    public void setTotalTests(int totalTests) { this.totalTests = totalTests; }
    public int getPassed() { return passed; }
    public void setPassed(int passed) { this.passed = passed; }
    public int getFailed() { return failed; }
    public void setFailed(int failed) { this.failed = failed; }
    public int getErrors() { return errors; }
    public void setErrors(int errors) { this.errors = errors; }
    public int getSkipped() { return skipped; }
    public void setSkipped(int skipped) { this.skipped = skipped; }
    public long getDuration() { return duration; }
    public void setDuration(long duration) { this.duration = duration; }
    public String getStartedAt() { return startedAt; }
    public void setStartedAt(String startedAt) { this.startedAt = startedAt; }
    public String getFinishedAt() { return finishedAt; }
    public void setFinishedAt(String finishedAt) { this.finishedAt = finishedAt; }
    public Map<String, Integer> getBySeverity() { return bySeverity; }
    public void setBySeverity(Map<String, Integer> bySeverity) { this.bySeverity = bySeverity; }
    public Map<String, Integer> getByCategory() { return byCategory; }
    public void setByCategory(Map<String, Integer> byCategory) { this.byCategory = byCategory; }
    public List<SingleTestResult> getResults() { return results; }
    public void setResults(List<SingleTestResult> results) { this.results = results; }

    /** Returns true if all tests passed (no failures or errors). */
    public boolean isSuccess() {
        return failed == 0 && errors == 0;
    }

    /** Returns the count of failures at or above the given severity. */
    public int failuresAtSeverity(String... severities) {
        if (bySeverity == null) return 0;
        int count = 0;
        for (String sev : severities) {
            count += bySeverity.getOrDefault(sev, 0);
        }
        return count;
    }
}
