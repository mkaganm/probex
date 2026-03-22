package io.probex.gradle;

import org.gradle.api.provider.ListProperty;
import org.gradle.api.provider.Property;

import java.util.Arrays;

/**
 * Configuration extension for the PROBEX Gradle plugin.
 *
 * <pre>
 * probex {
 *     targetUrl = "http://localhost:8080"
 *     serverUrl = "http://localhost:9712"
 *     failOn = listOf("critical", "high")
 *     categories = listOf("security", "happy_path")
 *     maxDepth = 3
 *     skip = false
 * }
 * </pre>
 */
public abstract class ProbexExtension {

    /** The base URL of the API to test. */
    public abstract Property<String> getTargetUrl();

    /** The PROBEX server URL. */
    public abstract Property<String> getServerUrl();

    /** Severity levels that cause build failure. */
    public abstract ListProperty<String> getFailOn();

    /** Test categories to run (empty = all). */
    public abstract ListProperty<String> getCategories();

    /** Scan depth. */
    public abstract Property<Integer> getMaxDepth();

    /** Skip PROBEX tests. */
    public abstract Property<Boolean> getSkip();

    public ProbexExtension() {
        getServerUrl().convention("http://localhost:9712");
        getFailOn().convention(Arrays.asList("critical", "high"));
        getCategories().convention(java.util.Collections.emptyList());
        getMaxDepth().convention(3);
        getSkip().convention(false);
    }
}
