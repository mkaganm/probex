package io.probex.gradle;

import org.gradle.api.Plugin;
import org.gradle.api.Project;

/**
 * PROBEX Gradle plugin for automated API testing.
 *
 * <pre>
 * plugins {
 *     id("io.probex") version "1.0.0"
 * }
 *
 * probex {
 *     targetUrl = "http://localhost:8080"
 *     serverUrl = "http://localhost:9712"
 *     failOn = listOf("critical", "high")
 *     maxDepth = 3
 * }
 * </pre>
 *
 * Tasks:
 * <ul>
 *   <li>{@code gradle probexScan} — Discover API endpoints</li>
 *   <li>{@code gradle probexTest} — Run API tests</li>
 *   <li>{@code gradle probexReport} — Generate test report</li>
 * </ul>
 */
public class ProbexPlugin implements Plugin<Project> {

    @Override
    public void apply(Project project) {
        // Register extension for configuration.
        ProbexExtension extension = project.getExtensions()
                .create("probex", ProbexExtension.class);

        // Register tasks.
        project.getTasks().register("probexScan", ProbexScanTask.class, task -> {
            task.setGroup("verification");
            task.setDescription("Discover API endpoints using PROBEX");
            task.getTargetUrl().convention(extension.getTargetUrl());
            task.getServerUrl().convention(extension.getServerUrl());
            task.getMaxDepth().convention(extension.getMaxDepth());
        });

        project.getTasks().register("probexTest", ProbexTestTask.class, task -> {
            task.setGroup("verification");
            task.setDescription("Run PROBEX API tests");
            task.getTargetUrl().convention(extension.getTargetUrl());
            task.getServerUrl().convention(extension.getServerUrl());
            task.getFailOn().convention(extension.getFailOn());
            task.getCategories().convention(extension.getCategories());
            task.getMaxDepth().convention(extension.getMaxDepth());
            // Depend on scan first.
            task.dependsOn("probexScan");
        });

        project.getTasks().register("probexReport", ProbexReportTask.class, task -> {
            task.setGroup("verification");
            task.setDescription("Generate PROBEX test report");
            task.getServerUrl().convention(extension.getServerUrl());
            task.getReportDir().convention(
                    project.getLayout().getBuildDirectory().dir("reports/probex"));
            task.dependsOn("probexTest");
        });
    }
}
