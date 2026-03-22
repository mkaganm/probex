package io.probex;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.probex.models.*;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

/**
 * Java client for the PROBEX REST API server.
 *
 * <pre>{@code
 * var client = new ProbexClient("http://localhost:9712");
 * var profile = client.scan(new ScanRequest("https://api.example.com"));
 * var results = client.run(new RunRequest());
 * System.out.println("Passed: " + results.getPassed());
 * }</pre>
 */
public class ProbexClient {

    private final String baseUrl;
    private final HttpClient httpClient;
    private final ObjectMapper mapper;

    public ProbexClient() {
        this("http://localhost:9712");
    }

    public ProbexClient(String baseUrl) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
        this.mapper = new ObjectMapper()
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    /** Check server health. */
    public HealthResponse health() throws ProbexException {
        return doGet("/api/v1/health", HealthResponse.class);
    }

    /** Get the current API profile. */
    public ScanResult getProfile() throws ProbexException {
        return doGet("/api/v1/profile", ScanResult.class);
    }

    /** Scan an API and return the discovered profile. */
    public ScanResult scan(ScanRequest request) throws ProbexException {
        return doPost("/api/v1/scan", request, ScanResult.class);
    }

    /** Run tests and return results. */
    public TestResult run(RunRequest request) throws ProbexException {
        return doPost("/api/v1/run", request, TestResult.class);
    }

    /** Run tests with default options. */
    public TestResult run() throws ProbexException {
        return run(new RunRequest());
    }

    /** Get the latest test results. */
    public TestResult getResults() throws ProbexException {
        return doGet("/api/v1/results", TestResult.class);
    }

    private <T> T doGet(String path, Class<T> responseType) throws ProbexException {
        try {
            var request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .timeout(Duration.ofSeconds(120))
                    .GET()
                    .build();
            var response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() >= 400) {
                throw new ProbexException("GET " + path + " returned " + response.statusCode() + ": " + response.body());
            }
            return mapper.readValue(response.body(), responseType);
        } catch (ProbexException e) {
            throw e;
        } catch (IOException | InterruptedException e) {
            throw new ProbexException("GET " + path + " failed: " + e.getMessage(), e);
        }
    }

    private <T> T doPost(String path, Object body, Class<T> responseType) throws ProbexException {
        try {
            var json = mapper.writeValueAsString(body);
            var request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .timeout(Duration.ofSeconds(120))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();
            var response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() >= 400) {
                throw new ProbexException("POST " + path + " returned " + response.statusCode() + ": " + response.body());
            }
            return mapper.readValue(response.body(), responseType);
        } catch (ProbexException e) {
            throw e;
        } catch (IOException | InterruptedException e) {
            throw new ProbexException("POST " + path + " failed: " + e.getMessage(), e);
        }
    }
}
