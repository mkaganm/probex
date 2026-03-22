package io.probex.gradle;

import io.probex.ProbexClient;
import io.probex.ProbexException;
import io.probex.models.TestResult;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import org.gradle.api.DefaultTask;
import org.gradle.api.GradleException;
import org.gradle.api.file.DirectoryProperty;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.Optional;
import org.gradle.api.tasks.OutputDirectory;
import org.gradle.api.tasks.TaskAction;

import java.io.File;
import java.io.IOException;

/**
 * Gradle task that generates a PROBEX test report.
 *
 * Usage: {@code gradle probexReport}
 */
public abstract class ProbexReportTask extends DefaultTask {

    @Input
    @Optional
    public abstract Property<String> getServerUrl();

    @OutputDirectory
    public abstract DirectoryProperty getReportDir();

    @TaskAction
    public void report() {
        String server = getServerUrl().getOrElse("http://localhost:9712");

        var client = new ProbexClient(server);

        getLogger().lifecycle("PROBEX: Fetching results...");

        TestResult result;
        try {
            result = client.run(new io.probex.models.RunRequest());
        } catch (ProbexException e) {
            throw new GradleException("PROBEX: Failed to get results: " + e.getMessage(), e);
        }

        // Write JSON report.
        File reportDir = getReportDir().get().getAsFile();
        reportDir.mkdirs();

        File jsonReport = new File(reportDir, "probex-results.json");
        ObjectMapper mapper = new ObjectMapper();
        mapper.enable(SerializationFeature.INDENT_OUTPUT);

        try {
            mapper.writeValue(jsonReport, result);
            getLogger().lifecycle("PROBEX: Report written to {}", jsonReport.getAbsolutePath());
        } catch (IOException e) {
            throw new GradleException("PROBEX: Failed to write report: " + e.getMessage(), e);
        }

        // Write summary.
        File summaryFile = new File(reportDir, "probex-summary.txt");
        String summary = String.format(
                "PROBEX Test Report\n" +
                "==================\n" +
                "Total:   %d\n" +
                "Passed:  %d\n" +
                "Failed:  %d\n" +
                "Errors:  %d\n",
                result.getTotalTests(), result.getPassed(),
                result.getFailed(), result.getErrors());

        try {
            java.nio.file.Files.writeString(summaryFile.toPath(), summary);
        } catch (IOException e) {
            getLogger().warn("PROBEX: Could not write summary: {}", e.getMessage());
        }
    }
}
