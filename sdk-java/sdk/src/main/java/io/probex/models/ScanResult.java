package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class ScanResult {
    private String id;
    private String name;

    @JsonProperty("base_url")
    private String baseUrl;

    private List<Endpoint> endpoints;

    public ScanResult() {}

    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getBaseUrl() { return baseUrl; }
    public void setBaseUrl(String baseUrl) { this.baseUrl = baseUrl; }
    public List<Endpoint> getEndpoints() { return endpoints; }
    public void setEndpoints(List<Endpoint> endpoints) { this.endpoints = endpoints; }

    public int getEndpointCount() {
        return endpoints != null ? endpoints.size() : 0;
    }
}
