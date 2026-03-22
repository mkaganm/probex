package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class Endpoint {
    private String id;
    private String method;
    private String path;

    @JsonProperty("base_url")
    private String baseUrl;

    private List<String> tags;

    public Endpoint() {}

    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public String getMethod() { return method; }
    public void setMethod(String method) { this.method = method; }
    public String getPath() { return path; }
    public void setPath(String path) { this.path = path; }
    public String getBaseUrl() { return baseUrl; }
    public void setBaseUrl(String baseUrl) { this.baseUrl = baseUrl; }
    public List<String> getTags() { return tags; }
    public void setTags(List<String> tags) { this.tags = tags; }

    @Override
    public String toString() {
        return method + " " + path;
    }
}
