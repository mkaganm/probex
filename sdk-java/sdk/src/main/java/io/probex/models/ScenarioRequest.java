package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class ScenarioRequest {
    private List<Endpoint> endpoints;
    @JsonProperty("max_scenarios")
    private int maxScenarios = 10;

    public ScenarioRequest() {}

    public ScenarioRequest(List<Endpoint> endpoints) {
        this.endpoints = endpoints;
    }

    public ScenarioRequest(List<Endpoint> endpoints, int maxScenarios) {
        this.endpoints = endpoints;
        this.maxScenarios = maxScenarios;
    }

    public List<Endpoint> getEndpoints() { return endpoints; }
    public void setEndpoints(List<Endpoint> endpoints) { this.endpoints = endpoints; }
    public int getMaxScenarios() { return maxScenarios; }
    public void setMaxScenarios(int maxScenarios) { this.maxScenarios = maxScenarios; }
}
