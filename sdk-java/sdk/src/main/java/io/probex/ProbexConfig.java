package io.probex;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

/**
 * Configuration annotation for PROBEX JUnit 5 tests.
 *
 * <pre>{@code
 * @ExtendWith(ProbexExtension.class)
 * @ProbexConfig(baseUrl = "http://localhost:8080")
 * class ApiTests {
 *     @ProbexTest
 *     void allEndpointsShouldPass(ProbexResult result) {
 *         assertThat(result.getFailed()).isZero();
 *     }
 * }
 * }</pre>
 */
@Target(ElementType.TYPE)
@Retention(RetentionPolicy.RUNTIME)
public @interface ProbexConfig {
    /** Base URL of the API to test. */
    String baseUrl() default "";

    /** PROBEX server URL. */
    String serverUrl() default "http://localhost:9712";

    /** Severity levels that cause test failure. */
    String[] failOn() default {"critical", "high"};

    /** Test categories to run. Empty means all. */
    String[] categories() default {};
}
