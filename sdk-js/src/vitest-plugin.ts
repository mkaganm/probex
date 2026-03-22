import { expect } from 'vitest';
import { ProbexClient } from './client.js';
import type { RunSummary, ProbexConfig, RunRequest } from './types.js';

export interface ProbexVitestConfig extends ProbexConfig {
  categories?: string[];
}

/**
 * Run PROBEX tests and return the summary for use in Vitest assertions.
 */
export async function runProbex(config: ProbexVitestConfig = {}): Promise<RunSummary> {
  const client = new ProbexClient(config);
  const request: RunRequest = {};
  if (config.categories) {
    request.categories = config.categories;
  }
  return client.run(request);
}

/**
 * Custom Vitest matcher: expect(results).toProbexPass()
 * Passes when there are zero failures and zero errors.
 */
expect.extend({
  toProbexPass(received: RunSummary) {
    const pass = received.failed === 0 && received.errors === 0;
    return {
      pass,
      message: () =>
        pass
          ? `Expected probex tests to fail but all ${received.total_tests} passed`
          : `${received.failed} failed, ${received.errors} errors out of ${received.total_tests} tests`,
    };
  },
});
