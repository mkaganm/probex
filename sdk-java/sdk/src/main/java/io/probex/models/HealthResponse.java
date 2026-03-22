package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class HealthResponse {
    private String status;
    private String version;

    public HealthResponse() {}

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    @JsonProperty("version")
    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }
}
