package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class ScenarioResponse {
    private List<GeneratedTestCase> scenarios;
    @JsonProperty("model_used")
    private String modelUsed;
    @JsonProperty("tokens_used")
    private int tokensUsed;

    public List<GeneratedTestCase> getScenarios() { return scenarios; }
    public void setScenarios(List<GeneratedTestCase> scenarios) { this.scenarios = scenarios; }
    public String getModelUsed() { return modelUsed; }
    public void setModelUsed(String modelUsed) { this.modelUsed = modelUsed; }
    public int getTokensUsed() { return tokensUsed; }
    public void setTokensUsed(int tokensUsed) { this.tokensUsed = tokensUsed; }
}
