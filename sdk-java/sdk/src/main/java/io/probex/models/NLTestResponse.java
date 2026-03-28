package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class NLTestResponse {
    @JsonProperty("test_cases")
    private List<GeneratedTestCase> testCases;
    @JsonProperty("model_used")
    private String modelUsed;
    @JsonProperty("tokens_used")
    private int tokensUsed;

    public List<GeneratedTestCase> getTestCases() { return testCases; }
    public void setTestCases(List<GeneratedTestCase> testCases) { this.testCases = testCases; }
    public String getModelUsed() { return modelUsed; }
    public void setModelUsed(String modelUsed) { this.modelUsed = modelUsed; }
    public int getTokensUsed() { return tokensUsed; }
    public void setTokensUsed(int tokensUsed) { this.tokensUsed = tokensUsed; }
}
