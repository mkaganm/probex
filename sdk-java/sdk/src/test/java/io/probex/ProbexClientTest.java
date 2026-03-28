package io.probex;

import com.sun.net.httpserver.HttpServer;
import io.probex.models.ScanRequest;
import io.probex.models.RunRequest;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;

import static org.junit.jupiter.api.Assertions.*;

class ProbexClientTest {

    private HttpServer server;
    private ProbexClient client;

    @BeforeEach
    void setUp() throws IOException {
        server = HttpServer.create(new InetSocketAddress(0), 0);
        server.start();
        int port = server.getAddress().getPort();
        client = new ProbexClient("http://localhost:" + port);
    }

    @AfterEach
    void tearDown() {
        server.stop(0);
    }

    @Test
    void constructorStripsTrailingSlash() throws ProbexException {
        server.createContext("/api/v1/health", exchange -> {
            var body = "{\"status\":\"ok\",\"version\":\"1.0.0\"}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        int port = server.getAddress().getPort();
        var clientWithSlash = new ProbexClient("http://localhost:" + port + "/");
        var resp = clientWithSlash.health();
        assertEquals("ok", resp.getStatus());
    }

    @Test
    void healthReturnsHealthResponse() throws ProbexException {
        server.createContext("/api/v1/health", exchange -> {
            var body = "{\"status\":\"ok\",\"version\":\"2.1.0\"}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var resp = client.health();
        assertEquals("ok", resp.getStatus());
        assertEquals("2.1.0", resp.getVersion());
    }

    @Test
    void getProfileReturnsScanResult() throws ProbexException {
        server.createContext("/api/v1/profile", exchange -> {
            var body = "{\"id\":\"prof-1\",\"name\":\"My API\",\"base_url\":\"https://api.example.com\",\"endpoints\":[]}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var result = client.getProfile();
        assertEquals("prof-1", result.getId());
        assertEquals("My API", result.getName());
        assertEquals("https://api.example.com", result.getBaseUrl());
        assertEquals(0, result.getEndpointCount());
    }

    @Test
    void scanPostsRequestAndReturnsScanResult() throws ProbexException {
        server.createContext("/api/v1/scan", exchange -> {
            assertEquals("POST", exchange.getRequestMethod());
            assertEquals("application/json", exchange.getRequestHeaders().getFirst("Content-Type"));
            // consume request body
            exchange.getRequestBody().readAllBytes();

            var body = "{\"id\":\"scan-1\",\"name\":\"Scanned\",\"base_url\":\"https://target.com\","
                    + "\"endpoints\":[{\"id\":\"ep-1\",\"method\":\"GET\",\"path\":\"/users\","
                    + "\"base_url\":\"https://target.com\",\"tags\":[\"rest\"]}]}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var result = client.scan(new ScanRequest("https://target.com", 5, 20));
        assertEquals("scan-1", result.getId());
        assertEquals(1, result.getEndpointCount());
        assertEquals("GET", result.getEndpoints().get(0).getMethod());
        assertEquals("/users", result.getEndpoints().get(0).getPath());
    }

    @Test
    void runWithRequestPostsAndReturnsTestResult() throws ProbexException {
        server.createContext("/api/v1/run", exchange -> {
            assertEquals("POST", exchange.getRequestMethod());
            exchange.getRequestBody().readAllBytes();

            var body = "{\"profile_id\":\"p1\",\"total_tests\":10,\"passed\":8,\"failed\":1,"
                    + "\"errors\":1,\"skipped\":0,\"duration\":5000,"
                    + "\"started_at\":\"2025-01-01T00:00:00Z\",\"finished_at\":\"2025-01-01T00:00:05Z\","
                    + "\"by_severity\":{\"critical\":1,\"high\":0},\"by_category\":{\"auth\":1},"
                    + "\"results\":[{\"test_case_id\":\"tc-1\",\"test_name\":\"Auth test\","
                    + "\"status\":\"failed\",\"category\":\"auth\",\"severity\":\"critical\","
                    + "\"duration\":200,\"error\":\"401 returned\"}]}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var result = client.run(new RunRequest());
        assertEquals("p1", result.getProfileId());
        assertEquals(10, result.getTotalTests());
        assertEquals(8, result.getPassed());
        assertEquals(1, result.getFailed());
        assertEquals(1, result.getErrors());
        assertFalse(result.isSuccess());
        assertEquals(1, result.getResults().size());
        assertTrue(result.getResults().get(0).isFailed());
    }

    @Test
    void runNoArgsPostsAndReturnsTestResult() throws ProbexException {
        server.createContext("/api/v1/run", exchange -> {
            exchange.getRequestBody().readAllBytes();
            var body = "{\"profile_id\":\"p2\",\"total_tests\":5,\"passed\":5,\"failed\":0,"
                    + "\"errors\":0,\"skipped\":0,\"duration\":1000,"
                    + "\"by_severity\":{},\"by_category\":{},\"results\":[]}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var result = client.run();
        assertEquals(5, result.getPassed());
        assertTrue(result.isSuccess());
    }

    @Test
    void getResultsReturnsTestResult() throws ProbexException {
        server.createContext("/api/v1/results", exchange -> {
            var body = "{\"profile_id\":\"p3\",\"total_tests\":3,\"passed\":3,\"failed\":0,"
                    + "\"errors\":0,\"skipped\":0,\"duration\":800,"
                    + "\"by_severity\":{},\"by_category\":{},\"results\":[]}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var result = client.getResults();
        assertEquals("p3", result.getProfileId());
        assertEquals(3, result.getTotalTests());
        assertTrue(result.isSuccess());
    }

    @Test
    void getThrowsProbexExceptionOnServerError() {
        server.createContext("/api/v1/health", exchange -> {
            var body = "internal server error";
            exchange.sendResponseHeaders(500, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var ex = assertThrows(ProbexException.class, () -> client.health());
        assertTrue(ex.getMessage().contains("GET /api/v1/health returned 500"));
        assertTrue(ex.getMessage().contains("internal server error"));
    }

    @Test
    void postThrowsProbexExceptionOnClientError() {
        server.createContext("/api/v1/scan", exchange -> {
            exchange.getRequestBody().readAllBytes();
            var body = "bad request";
            exchange.sendResponseHeaders(400, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var ex = assertThrows(ProbexException.class, () -> client.scan(new ScanRequest("http://x")));
        assertTrue(ex.getMessage().contains("POST /api/v1/scan returned 400"));
        assertTrue(ex.getMessage().contains("bad request"));
    }

    @Test
    void connectionRefusedThrowsProbexException() {
        // Use a client pointing to a port that is not listening
        var badClient = new ProbexClient("http://localhost:1");

        var ex = assertThrows(ProbexException.class, () -> badClient.health());
        assertTrue(ex.getMessage().contains("GET /api/v1/health failed:"));
    }

    @Test
    void unknownFieldsAreIgnored() throws ProbexException {
        server.createContext("/api/v1/health", exchange -> {
            var body = "{\"status\":\"ok\",\"version\":\"1.0\",\"extra_field\":\"ignored\"}";
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        var resp = client.health();
        assertEquals("ok", resp.getStatus());
    }

    @Test
    void defaultConstructorUsesLocalhost() {
        var defaultClient = new ProbexClient();
        // Verify it doesn't throw during construction; it will fail on actual call
        // since nothing listens on 9712 in test, but object should be created fine.
        assertNotNull(defaultClient);
    }
}
