package io.probex.gradle;

import io.probex.ProbexClient;
import io.probex.ProbexException;
import io.probex.models.ScanRequest;
import org.gradle.api.DefaultTask;
import org.gradle.api.GradleException;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.Optional;
import org.gradle.api.tasks.TaskAction;

/**
 * Gradle task that scans an API for endpoint discovery.
 *
 * Usage: {@code gradle probexScan}
 */
public abstract class ProbexScanTask extends DefaultTask {

    @Input
    public abstract Property<String> getTargetUrl();

    @Input
    @Optional
    public abstract Property<String> getServerUrl();

    @Input
    @Optional
    public abstract Property<Integer> getMaxDepth();

    @TaskAction
    public void scan() {
        String target = getTargetUrl().getOrNull();
        if (target == null || target.isEmpty()) {
            throw new GradleException("PROBEX: targetUrl is required. Set it in the probex extension.");
        }

        String server = getServerUrl().getOrElse("http://localhost:9712");
        int depth = getMaxDepth().getOrElse(3);

        var client = new ProbexClient(server);

        getLogger().lifecycle("PROBEX: Scanning {}...", target);

        try {
            var result = client.scan(new ScanRequest(target, depth, 10));
            getLogger().lifecycle("PROBEX: Discovered {} endpoints", result.getEndpointCount());
        } catch (ProbexException e) {
            throw new GradleException("PROBEX scan failed: " + e.getMessage(), e);
        }
    }
}
