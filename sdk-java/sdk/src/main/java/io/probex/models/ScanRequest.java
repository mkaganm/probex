package io.probex.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ScanRequest {
    @JsonProperty("base_url")
    private String baseUrl;

    @JsonProperty("max_depth")
    private int maxDepth = 3;

    private int concurrency = 10;

    public ScanRequest() {}

    public ScanRequest(String baseUrl) {
        this.baseUrl = baseUrl;
    }

    public ScanRequest(String baseUrl, int maxDepth, int concurrency) {
        this.baseUrl = baseUrl;
        this.maxDepth = maxDepth;
        this.concurrency = concurrency;
    }

    public String getBaseUrl() { return baseUrl; }
    public void setBaseUrl(String baseUrl) { this.baseUrl = baseUrl; }
    public int getMaxDepth() { return maxDepth; }
    public void setMaxDepth(int maxDepth) { this.maxDepth = maxDepth; }
    public int getConcurrency() { return concurrency; }
    public void setConcurrency(int concurrency) { this.concurrency = concurrency; }
}
