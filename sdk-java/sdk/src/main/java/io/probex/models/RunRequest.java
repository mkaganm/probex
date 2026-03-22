package io.probex.models;

import java.util.List;

public class RunRequest {
    private List<String> categories;
    private int concurrency = 5;
    private int timeout = 30;

    public RunRequest() {}

    public RunRequest(List<String> categories) {
        this.categories = categories;
    }

    public List<String> getCategories() { return categories; }
    public void setCategories(List<String> categories) { this.categories = categories; }
    public int getConcurrency() { return concurrency; }
    public void setConcurrency(int concurrency) { this.concurrency = concurrency; }
    public int getTimeout() { return timeout; }
    public void setTimeout(int timeout) { this.timeout = timeout; }
}
