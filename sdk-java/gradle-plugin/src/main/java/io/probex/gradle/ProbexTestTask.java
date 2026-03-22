package io.probex.gradle;

import io.probex.ProbexClient;
import io.probex.ProbexException;
import io.probex.models.RunRequest;
import io.probex.models.ScanRequest;
import io.probex.models.TestResult;
import org.gradle.api.DefaultTask;
import org.gradle.api.GradleException;
import org.gradle.api.provider.ListProperty;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.Optional;
import org.gradle.api.tasks.TaskAction;

import java.util.List;

/**
 * Gradle task that runs PROBEX API tests.
 *
 * Usage: {@code gradle probexTest}
 */
public abstract class ProbexTestTask extends DefaultTask {

    @Input
    public abstract Property<String> getTargetUrl();

    @Input
    @Optional
    public abstract Property<String> getServerUrl();

    @Input
    @Optional
    public abstract ListProperty<String> getFailOn();

    @Input
    @Optional
    public abstract ListProperty<String> getCategories();

    @Input
    @Optional
    public abstract Property<Integer> getMaxDepth();

    @TaskAction
    public void test() {
        String target = getTargetUrl().getOrNull();
        if (target == null || target.isEmpty()) {
            throw new GradleException("PROBEX: targetUrl is required.");
        }

        String server = getServerUrl().getOrElse("http://localhost:9712");
        List<String> failOnSeverities = getFailOn().getOrElse(List.of("critical", "high"));
        List<String> cats = getCategories().getOrElse(List.of());
        int depth = getMaxDepth().getOrElse(3);

        var client = new ProbexClient(server);

        // Scan.
        getLogger().lifecycle("PROBEX: Scanning {}...", target);
        try {
            var scanResult = client.scan(new ScanRequest(target, depth, 10));
            getLogger().lifecycle("PROBEX: Discovered {} endpoints", scanResult.getEndpointCount());
        } catch (ProbexException e) {
            throw new GradleException("PROBEX scan failed: " + e.getMessage(), e);
        }

        // Run.
        getLogger().lifecycle("PROBEX: Running tests...");
        var runRequest = new RunRequest();
        if (!cats.isEmpty()) {
            runRequest.setCategories(cats);
        }

        TestResult result;
        try {
            result = client.run(runRequest);
        } catch (ProbexException e) {
            throw new GradleException("PROBEX run failed: " + e.getMessage(), e);
        }

        // Report.
        getLogger().lifecycle("PROBEX: {} tests — {} passed, {} failed, {} errors",
                result.getTotalTests(), result.getPassed(), result.getFailed(), result.getErrors());

        // Check fail-on.
        if (!failOnSeverities.isEmpty()) {
            int failCount = result.failuresAtSeverity(failOnSeverities.toArray(new String[0]));
            if (failCount > 0) {
                throw new GradleException(String.format(
                        "PROBEX: %d findings at severity %s", failCount, failOnSeverities));
            }
        }

        getLogger().lifecycle("PROBEX: All checks passed");
    }
}
