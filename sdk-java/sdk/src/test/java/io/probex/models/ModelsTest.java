package io.probex.models;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class ModelsTest {

    private ObjectMapper mapper;

    @BeforeEach
    void setUp() {
        mapper = new ObjectMapper()
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    // --- HealthResponse ---

    @Test
    void healthResponseRoundTrip() throws Exception {
        var json = "{\"status\":\"ok\",\"version\":\"2.0.1\"}";
        var obj = mapper.readValue(json, HealthResponse.class);
        assertEquals("ok", obj.getStatus());
        assertEquals("2.0.1", obj.getVersion());

        var serialized = mapper.writeValueAsString(obj);
        var deserialized = mapper.readValue(serialized, HealthResponse.class);
        assertEquals("ok", deserialized.getStatus());
        assertEquals("2.0.1", deserialized.getVersion());
    }

    // --- Endpoint ---

    @Test
    void endpointRoundTrip() throws Exception {
        var json = "{\"id\":\"ep-1\",\"method\":\"POST\",\"path\":\"/users\","
                + "\"base_url\":\"https://api.example.com\",\"tags\":[\"rest\",\"v2\"]}";
        var obj = mapper.readValue(json, Endpoint.class);
        assertEquals("ep-1", obj.getId());
        assertEquals("POST", obj.getMethod());
        assertEquals("/users", obj.getPath());
        assertEquals("https://api.example.com", obj.getBaseUrl());
        assertEquals(List.of("rest", "v2"), obj.getTags());
        assertEquals("POST /users", obj.toString());

        var serialized = mapper.writeValueAsString(obj);
        assertTrue(serialized.contains("\"base_url\""));
        var deserialized = mapper.readValue(serialized, Endpoint.class);
        assertEquals("https://api.example.com", deserialized.getBaseUrl());
    }

    // --- ScanRequest ---

    @Test
    void scanRequestDefaultConstructor() throws Exception {
        var req = new ScanRequest();
        var json = mapper.writeValueAsString(req);
        var deserialized = mapper.readValue(json, ScanRequest.class);
        assertNull(deserialized.getBaseUrl());
        assertEquals(3, deserialized.getMaxDepth());
        assertEquals(10, deserialized.getConcurrency());
    }

    @Test
    void scanRequestBaseUrlConstructor() throws Exception {
        var req = new ScanRequest("https://target.com");
        assertEquals("https://target.com", req.getBaseUrl());
        assertEquals(3, req.getMaxDepth());
        assertEquals(10, req.getConcurrency());

        var json = mapper.writeValueAsString(req);
        assertTrue(json.contains("\"base_url\""));
        assertTrue(json.contains("\"max_depth\""));
    }

    @Test
    void scanRequestFullConstructor() throws Exception {
        var req = new ScanRequest("https://target.com", 5, 20);
        var json = mapper.writeValueAsString(req);
        var deserialized = mapper.readValue(json, ScanRequest.class);
        assertEquals("https://target.com", deserialized.getBaseUrl());
        assertEquals(5, deserialized.getMaxDepth());
        assertEquals(20, deserialized.getConcurrency());
    }

    // --- ScanResult ---

    @Test
    void scanResultRoundTrip() throws Exception {
        var json = "{\"id\":\"sr-1\",\"name\":\"API Profile\",\"base_url\":\"https://api.test.com\","
                + "\"endpoints\":[{\"id\":\"e1\",\"method\":\"GET\",\"path\":\"/health\","
                + "\"base_url\":\"https://api.test.com\",\"tags\":[]}]}";
        var obj = mapper.readValue(json, ScanResult.class);
        assertEquals("sr-1", obj.getId());
        assertEquals("API Profile", obj.getName());
        assertEquals("https://api.test.com", obj.getBaseUrl());
        assertEquals(1, obj.getEndpointCount());
        assertEquals("GET", obj.getEndpoints().get(0).getMethod());

        var serialized = mapper.writeValueAsString(obj);
        assertTrue(serialized.contains("\"base_url\""));
        var deserialized = mapper.readValue(serialized, ScanResult.class);
        assertEquals(1, deserialized.getEndpointCount());
    }

    @Test
    void scanResultEndpointCountWithNullEndpoints() {
        var result = new ScanResult();
        assertEquals(0, result.getEndpointCount());
    }

    // --- SingleTestResult ---

    @Test
    void singleTestResultRoundTrip() throws Exception {
        var json = "{\"test_case_id\":\"tc-1\",\"test_name\":\"SQL Injection\","
                + "\"status\":\"passed\",\"category\":\"injection\",\"severity\":\"critical\","
                + "\"duration\":150,\"error\":null}";
        var obj = mapper.readValue(json, SingleTestResult.class);
        assertEquals("tc-1", obj.getTestCaseId());
        assertEquals("SQL Injection", obj.getTestName());
        assertEquals("passed", obj.getStatus());
        assertEquals("injection", obj.getCategory());
        assertEquals("critical", obj.getSeverity());
        assertEquals(150, obj.getDuration());
        assertNull(obj.getError());
        assertTrue(obj.isPassed());
        assertFalse(obj.isFailed());
    }

    @Test
    void singleTestResultFailedStatus() throws Exception {
        var json = "{\"test_case_id\":\"tc-2\",\"test_name\":\"Auth Bypass\","
                + "\"status\":\"failed\",\"category\":\"auth\",\"severity\":\"high\","
                + "\"duration\":200,\"error\":\"expected 403 got 200\"}";
        var obj = mapper.readValue(json, SingleTestResult.class);
        assertTrue(obj.isFailed());
        assertFalse(obj.isPassed());
        assertEquals("expected 403 got 200", obj.getError());
    }

    // --- TestResult ---

    @Test
    void testResultRoundTrip() throws Exception {
        var json = "{\"profile_id\":\"p-1\",\"total_tests\":20,\"passed\":17,\"failed\":2,"
                + "\"errors\":1,\"skipped\":0,\"duration\":12000,"
                + "\"started_at\":\"2025-06-01T10:00:00Z\",\"finished_at\":\"2025-06-01T10:00:12Z\","
                + "\"by_severity\":{\"critical\":1,\"high\":1,\"medium\":0},"
                + "\"by_category\":{\"auth\":1,\"injection\":1},"
                + "\"results\":[{\"test_case_id\":\"tc-1\",\"test_name\":\"Test A\","
                + "\"status\":\"failed\",\"category\":\"auth\",\"severity\":\"critical\","
                + "\"duration\":100,\"error\":\"fail\"}]}";
        var obj = mapper.readValue(json, TestResult.class);

        assertEquals("p-1", obj.getProfileId());
        assertEquals(20, obj.getTotalTests());
        assertEquals(17, obj.getPassed());
        assertEquals(2, obj.getFailed());
        assertEquals(1, obj.getErrors());
        assertEquals(0, obj.getSkipped());
        assertEquals(12000, obj.getDuration());
        assertEquals("2025-06-01T10:00:00Z", obj.getStartedAt());
        assertEquals("2025-06-01T10:00:12Z", obj.getFinishedAt());
        assertEquals(Map.of("critical", 1, "high", 1, "medium", 0), obj.getBySeverity());
        assertEquals(Map.of("auth", 1, "injection", 1), obj.getByCategory());
        assertEquals(1, obj.getResults().size());

        var serialized = mapper.writeValueAsString(obj);
        assertTrue(serialized.contains("\"profile_id\""));
        assertTrue(serialized.contains("\"total_tests\""));
        assertTrue(serialized.contains("\"by_severity\""));
    }

    @Test
    void testResultIsSuccessTrue() {
        var result = new TestResult();
        result.setFailed(0);
        result.setErrors(0);
        assertTrue(result.isSuccess());
    }

    @Test
    void testResultIsSuccessFalseWithFailures() {
        var result = new TestResult();
        result.setFailed(1);
        result.setErrors(0);
        assertFalse(result.isSuccess());
    }

    @Test
    void testResultIsSuccessFalseWithErrors() {
        var result = new TestResult();
        result.setFailed(0);
        result.setErrors(2);
        assertFalse(result.isSuccess());
    }

    @Test
    void testResultFailuresAtSeverity() {
        var result = new TestResult();
        result.setBySeverity(Map.of("critical", 3, "high", 2, "medium", 5));

        assertEquals(3, result.failuresAtSeverity("critical"));
        assertEquals(5, result.failuresAtSeverity("critical", "high"));
        assertEquals(10, result.failuresAtSeverity("critical", "high", "medium"));
        assertEquals(0, result.failuresAtSeverity("low"));
    }

    @Test
    void testResultFailuresAtSeverityWithNullMap() {
        var result = new TestResult();
        assertEquals(0, result.failuresAtSeverity("critical"));
    }
}
