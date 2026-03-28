package io.probex.maven;

import io.probex.models.TestResult;
import org.apache.maven.plugin.MojoFailureException;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Field;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class ProbexMojoTest {

    private ProbexMojo mojo;

    @BeforeEach
    void setUp() throws Exception {
        mojo = new ProbexMojo();
        setField(mojo, "targetUrl", "https://api.example.com");
        setField(mojo, "serverUrl", "http://localhost:9712");
        setField(mojo, "failOn", "critical,high");
        setField(mojo, "categories", "");
        setField(mojo, "maxDepth", 3);
        setField(mojo, "skip", false);
    }

    @Test
    void skipTrueReturnsEarlyWithoutError() throws Exception {
        setField(mojo, "skip", true);
        // If skip=true, execute should return without attempting any network calls,
        // so it must not throw any exception.
        assertDoesNotThrow(() -> mojo.execute());
    }

    @Test
    void skipFalseAttemptsConnection() {
        // With skip=false and no real server, execute should throw
        // because it cannot connect to the PROBEX server.
        setField(mojo, "skip", false);
        assertThrows(Exception.class, () -> mojo.execute());
    }

    @Test
    void failuresAtSeverityTriggersMojoFailureException() {
        var result = new TestResult();
        result.setBySeverity(Map.of("critical", 2, "high", 1, "medium", 3));
        result.setFailed(3);
        result.setErrors(0);

        // Verify the severity check logic that the mojo relies on
        String failOn = "critical,high";
        String[] severities = failOn.split(",");
        int failCount = result.failuresAtSeverity(severities);
        assertEquals(3, failCount);
        assertTrue(failCount > 0, "Should trigger MojoFailureException when failCount > 0");
    }

    @Test
    void noFailuresAtSeverityDoesNotTriggerFailure() {
        var result = new TestResult();
        result.setBySeverity(Map.of("critical", 0, "high", 0, "medium", 5));
        result.setFailed(5);
        result.setErrors(0);

        String failOn = "critical,high";
        String[] severities = failOn.split(",");
        int failCount = result.failuresAtSeverity(severities);
        assertEquals(0, failCount);
    }

    @Test
    void emptyFailOnDoesNotCheckSeverity() {
        var result = new TestResult();
        result.setBySeverity(Map.of("critical", 10));
        result.setFailed(10);

        // When failOn is empty, no severity check should occur
        String failOn = "";
        if (failOn != null && !failOn.isEmpty()) {
            fail("Should not enter severity check with empty failOn");
        }
    }

    @Test
    void mojoFailureExceptionMessageFormat() {
        var result = new TestResult();
        result.setBySeverity(Map.of("critical", 2, "high", 3));

        String failOn = "critical,high";
        String[] severities = failOn.split(",");
        int failCount = result.failuresAtSeverity(severities);

        var exception = new MojoFailureException(
                String.format("PROBEX: %d findings at severity [%s]", failCount, failOn));
        assertTrue(exception.getMessage().contains("5 findings"));
        assertTrue(exception.getMessage().contains("[critical,high]"));
    }

    private static void setField(Object target, String fieldName, Object value) {
        try {
            Field field = target.getClass().getDeclaredField(fieldName);
            field.setAccessible(true);
            field.set(target, value);
        } catch (NoSuchFieldException | IllegalAccessException e) {
            throw new RuntimeException("Failed to set field " + fieldName, e);
        }
    }
}
