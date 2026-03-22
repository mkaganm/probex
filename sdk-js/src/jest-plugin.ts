import { ProbexClient } from './client.js';
import type { RunSummary, ProbexConfig, RunRequest } from './types.js';

export interface ProbexJestConfig extends ProbexConfig {
  failOn?: string[];
  categories?: string[];
}

declare global {
  namespace jest {
    interface Matchers<R> {
      toProbexPass(): R;
    }
  }
}

/**
 * Run PROBEX tests and return the summary for use in Jest assertions.
 */
export async function runProbex(config: ProbexJestConfig = {}): Promise<RunSummary> {
  const client = new ProbexClient(config);
  const request: RunRequest = {};
  if (config.categories) {
    request.categories = config.categories;
  }
  return client.run(request);
}

/**
 * Custom Jest matcher: expect(results).toProbexPass()
 * Passes when there are zero failures and zero errors.
 */
export const probexMatchers = {
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
};
