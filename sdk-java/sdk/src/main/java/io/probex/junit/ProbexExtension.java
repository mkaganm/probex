package io.probex.junit;

import io.probex.ProbexClient;
import io.probex.ProbexConfig;
import io.probex.ProbexException;
import io.probex.models.RunRequest;
import io.probex.models.ScanRequest;
import io.probex.models.TestResult;
import org.junit.jupiter.api.extension.*;

import java.util.Arrays;
import java.util.List;

/**
 * JUnit 5 extension that integrates PROBEX API testing.
 *
 * <pre>{@code
 * @ExtendWith(ProbexExtension.class)
 * @ProbexConfig(baseUrl = "http://localhost:8080", failOn = {"critical", "high"})
 * class ApiTests {
 *     @Test
 *     void allTestsShouldPass(TestResult result) {
 *         assertTrue(result.isSuccess());
 *     }
 * }
 * }</pre>
 */
public class ProbexExtension implements BeforeAllCallback, ParameterResolver {

    private static final ExtensionContext.Namespace NAMESPACE =
            ExtensionContext.Namespace.create(ProbexExtension.class);

    @Override
    public void beforeAll(ExtensionContext context) throws Exception {
        var testClass = context.getRequiredTestClass();
        var config = testClass.getAnnotation(ProbexConfig.class);

        String serverUrl = config != null ? config.serverUrl() : "http://localhost:9712";
        String baseUrl = config != null ? config.baseUrl() : "";
        String[] categories = config != null ? config.categories() : new String[]{};
        String[] failOn = config != null ? config.failOn() : new String[]{"critical", "high"};

        var client = new ProbexClient(serverUrl);

        // Scan if baseUrl is provided
        if (!baseUrl.isEmpty()) {
            try {
                client.scan(new ScanRequest(baseUrl));
            } catch (ProbexException e) {
                throw new ExtensionConfigurationException("PROBEX scan failed: " + e.getMessage(), e);
            }
        }

        // Run tests
        var request = new RunRequest();
        if (categories.length > 0) {
            request.setCategories(Arrays.asList(categories));
        }

        TestResult result;
        try {
            result = client.run(request);
        } catch (ProbexException e) {
            throw new ExtensionConfigurationException("PROBEX run failed: " + e.getMessage(), e);
        }

        // Store result for parameter injection
        context.getStore(NAMESPACE).put("result", result);
        context.getStore(NAMESPACE).put("failOn", List.of(failOn));
    }

    @Override
    public boolean supportsParameter(ParameterContext parameterContext, ExtensionContext extensionContext) {
        return parameterContext.getParameter().getType() == TestResult.class;
    }

    @Override
    public Object resolveParameter(ParameterContext parameterContext, ExtensionContext extensionContext) {
        return extensionContext.getStore(NAMESPACE).get("result", TestResult.class);
    }
}
