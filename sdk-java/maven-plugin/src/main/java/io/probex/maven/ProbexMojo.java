package io.probex.maven;

import io.probex.ProbexClient;
import io.probex.ProbexException;
import io.probex.models.RunRequest;
import io.probex.models.ScanRequest;
import io.probex.models.TestResult;
import org.apache.maven.plugin.AbstractMojo;
import org.apache.maven.plugin.MojoExecutionException;
import org.apache.maven.plugin.MojoFailureException;
import org.apache.maven.plugins.annotations.Mojo;
import org.apache.maven.plugins.annotations.Parameter;

import java.util.Arrays;
import java.util.List;

/**
 * Maven plugin goal that runs PROBEX API tests.
 *
 * <pre>{@code
 * <plugin>
 *     <groupId>io.probex</groupId>
 *     <artifactId>probex-maven-plugin</artifactId>
 *     <configuration>
 *         <targetUrl>${api.url}</targetUrl>
 *         <failOn>critical,high</failOn>
 *     </configuration>
 * </plugin>
 * }</pre>
 *
 * Usage: {@code mvn probex:test}
 */
@Mojo(name = "test")
public class ProbexMojo extends AbstractMojo {

    /** The base URL of the API to test. */
    @Parameter(property = "probex.targetUrl", required = true)
    private String targetUrl;

    /** The PROBEX server URL. */
    @Parameter(property = "probex.serverUrl", defaultValue = "http://localhost:9712")
    private String serverUrl;

    /** Comma-separated severity levels that cause build failure. */
    @Parameter(property = "probex.failOn", defaultValue = "critical,high")
    private String failOn;

    /** Comma-separated test categories to run. Empty means all. */
    @Parameter(property = "probex.categories", defaultValue = "")
    private String categories;

    /** Scan depth. */
    @Parameter(property = "probex.maxDepth", defaultValue = "3")
    private int maxDepth;

    /** Skip PROBEX tests. */
    @Parameter(property = "probex.skip", defaultValue = "false")
    private boolean skip;

    @Override
    public void execute() throws MojoExecutionException, MojoFailureException {
        if (skip) {
            getLog().info("PROBEX tests skipped");
            return;
        }

        var client = new ProbexClient(serverUrl);

        // Scan
        getLog().info("PROBEX: Scanning " + targetUrl + "...");
        try {
            var scanResult = client.scan(new ScanRequest(targetUrl, maxDepth, 10));
            getLog().info("PROBEX: Discovered " + scanResult.getEndpointCount() + " endpoints");
        } catch (ProbexException e) {
            throw new MojoExecutionException("PROBEX scan failed: " + e.getMessage(), e);
        }

        // Run
        getLog().info("PROBEX: Running tests...");
        var runRequest = new RunRequest();
        if (categories != null && !categories.isEmpty()) {
            runRequest.setCategories(Arrays.asList(categories.split(",")));
        }

        TestResult result;
        try {
            result = client.run(runRequest);
        } catch (ProbexException e) {
            throw new MojoExecutionException("PROBEX run failed: " + e.getMessage(), e);
        }

        // Report
        getLog().info(String.format("PROBEX: %d tests — %d passed, %d failed, %d errors",
                result.getTotalTests(), result.getPassed(), result.getFailed(), result.getErrors()));

        // Check fail-on
        if (failOn != null && !failOn.isEmpty()) {
            List<String> severities = Arrays.asList(failOn.split(","));
            int failCount = result.failuresAtSeverity(severities.toArray(new String[0]));
            if (failCount > 0) {
                throw new MojoFailureException(
                        String.format("PROBEX: %d findings at severity [%s]", failCount, failOn));
            }
        }

        getLog().info("PROBEX: All checks passed");
    }
}
