package io.probex.junit;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

import org.junit.jupiter.api.Test;

/**
 * Marks a method as a PROBEX test.
 * The method can receive a {@link io.probex.models.TestResult} parameter
 * containing the PROBEX test results.
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
@Test
public @interface ProbexTest {
}
