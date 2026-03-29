package io.probex.models;

import java.util.List;

public class NLTestRequest {
    private String description;
    private List<Endpoint> endpoints;

    public NLTestRequest() {}

    public NLTestRequest(String description) {
        this.description = description;
    }

    public NLTestRequest(String description, List<Endpoint> endpoints) {
        this.description = description;
        this.endpoints = endpoints;
    }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public List<Endpoint> getEndpoints() { return endpoints; }
    public void setEndpoints(List<Endpoint> endpoints) { this.endpoints = endpoints; }
}
