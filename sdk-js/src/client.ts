import type {
  ProbexConfig, ScanRequest, RunRequest, APIProfile, RunSummary,
  AIHealthResponse, ScenarioRequest, ScenarioResponse,
  SecurityAnalysisRequest, SecurityAnalysisResponse,
  NLTestRequest, NLTestResponse,
  AnomalyClassifyRequest, AnomalyClassifyResponse,
} from './types.js';

export class ProbexError extends Error {
  public readonly statusCode: number;
  public readonly responseBody: string;

  constructor(statusCode: number, responseBody: string) {
    super(`PROBEX API error (${statusCode}): ${responseBody}`);
    this.name = 'ProbexError';
    this.statusCode = statusCode;
    this.responseBody = responseBody;
  }
}

export class ProbexClient {
  private baseUrl: string;
  private timeout: number;

  constructor(config: ProbexConfig = {}) {
    this.baseUrl = config.serverUrl || 'http://localhost:9712';
    this.timeout = config.timeout || 30000;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    try {
      const init: RequestInit = {
        method,
        headers: { 'Content-Type': 'application/json' },
        signal: controller.signal,
      };

      if (body !== undefined) {
        init.body = JSON.stringify(body);
      }

      const response = await fetch(url, init);

      if (!response.ok) {
        const text = await response.text();
        throw new ProbexError(response.status, text);
      }

      return await response.json() as T;
    } finally {
      clearTimeout(timer);
    }
  }

  async health(): Promise<{ status: string; version: string }> {
    return this.request<{ status: string; version: string }>('GET', '/api/v1/health');
  }

  async scan(request: ScanRequest): Promise<APIProfile> {
    return this.request<APIProfile>('POST', '/api/v1/scan', request);
  }

  async run(request?: RunRequest): Promise<RunSummary> {
    return this.request<RunSummary>('POST', '/api/v1/run', request ?? {});
  }

  async getProfile(): Promise<APIProfile> {
    return this.request<APIProfile>('GET', '/api/v1/profile');
  }

  async getResults(): Promise<RunSummary> {
    return this.request<RunSummary>('GET', '/api/v1/results');
  }

  // --- AI endpoints ---

  async aiHealth(): Promise<AIHealthResponse> {
    return this.request<AIHealthResponse>('GET', '/api/v1/ai/health');
  }

  async aiScenarios(request: ScenarioRequest): Promise<ScenarioResponse> {
    return this.request<ScenarioResponse>('POST', '/api/v1/ai/scenarios', request);
  }

  async aiSecurity(request: SecurityAnalysisRequest): Promise<SecurityAnalysisResponse> {
    return this.request<SecurityAnalysisResponse>('POST', '/api/v1/ai/security', request);
  }

  async aiNLToTest(request: NLTestRequest): Promise<NLTestResponse> {
    return this.request<NLTestResponse>('POST', '/api/v1/ai/nl-to-test', request);
  }

  async aiAnomaly(request: AnomalyClassifyRequest): Promise<AnomalyClassifyResponse> {
    return this.request<AnomalyClassifyResponse>('POST', '/api/v1/ai/anomaly', request);
  }
}
