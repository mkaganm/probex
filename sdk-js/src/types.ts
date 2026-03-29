export interface Endpoint {
  id: string;
  method: string;
  path: string;
  base_url: string;
  query_params?: Parameter[];
  path_params?: Parameter[];
  request_body?: Schema;
  auth?: AuthInfo;
  tags?: string[];
}

export interface Parameter {
  name: string;
  type: string;
  required: boolean;
  example?: string;
}

export interface Schema {
  type: string;
  properties?: Record<string, Schema>;
  items?: Schema;
  required?: string[];
  format?: string;
}

export interface AuthInfo {
  type: string;
  location: string;
  key: string;
}

export interface TestCase {
  id: string;
  name: string;
  description?: string;
  category: string;
  severity: string;
  request: TestRequest;
  assertions: Assertion[];
  tags?: string[];
}

export interface TestRequest {
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: string;
  timeout?: number;
}

export interface Assertion {
  type: string;
  target: string;
  operator: string;
  expected: unknown;
}

export interface TestResult {
  test_case_id: string;
  test_name: string;
  status: 'passed' | 'failed' | 'error' | 'skipped';
  category: string;
  severity: string;
  duration: number;
  request: TestRequest;
  response?: TestResponse;
  assertions: AssertionResult[];
  error?: string;
}

export interface TestResponse {
  status_code: number;
  headers?: Record<string, string>;
  body?: string;
  duration: number;
}

export interface AssertionResult {
  assertion: Assertion;
  passed: boolean;
  actual?: unknown;
  message?: string;
}

export interface RunSummary {
  profile_id: string;
  total_tests: number;
  passed: number;
  failed: number;
  errors: number;
  skipped: number;
  duration: number;
  results: TestResult[];
  started_at: string;
  finished_at: string;
}

export interface ScanRequest {
  base_url: string;
  max_depth?: number;
  concurrency?: number;
}

export interface RunRequest {
  categories?: string[];
  concurrency?: number;
  timeout?: number;
  use_ai?: boolean;
}

export interface APIProfile {
  id: string;
  name: string;
  base_url: string;
  endpoints: Endpoint[];
}

export interface ProbexConfig {
  serverUrl?: string;
  timeout?: number;
}

// --- AI types ---

export interface AIHealthResponse {
  status: string;
  version: string;
  ai_mode: string;
  model: string;
}

export interface EndpointInfo {
  method: string;
  path: string;
  base_url: string;
  query_params?: { name: string; type: string; required: boolean }[];
  path_params?: { name: string; type: string; required: boolean }[];
  request_body?: Schema;
  auth?: { type: string; location: string; key: string };
  tags?: string[];
}

export interface ScenarioRequest {
  endpoints: EndpointInfo[];
  max_scenarios?: number;
}

export interface ScenarioResponse {
  scenarios: GeneratedTestCase[];
  model_used: string;
  tokens_used: number;
}

export interface GeneratedTestCase {
  name: string;
  description: string;
  category: string;
  severity: string;
  request: TestRequest;
  assertions: Assertion[];
  tags?: string[];
}

export interface SecurityAnalysisRequest {
  endpoints: EndpointInfo[];
  depth?: 'quick' | 'standard' | 'deep';
}

export interface SecurityFinding {
  title: string;
  description: string;
  severity: string;
  category: string;
  endpoint: string;
  evidence?: string;
  remediation?: string;
}

export interface SecurityAnalysisResponse {
  findings: SecurityFinding[];
  model_used: string;
  tokens_used: number;
}

export interface NLTestRequest {
  description: string;
  endpoints?: EndpointInfo[];
}

export interface NLTestResponse {
  test_cases: GeneratedTestCase[];
  model_used: string;
  tokens_used: number;
}

export interface AnomalyClassifyRequest {
  endpoint_id: string;
  observed_status: number;
  expected_status: number;
  response_body?: string;
  response_time_ms?: number;
  baseline_time_ms?: number;
}

export interface AnomalyClassifyResponse {
  classification: string;
  confidence: number;
  explanation: string;
  severity: string;
  model_used: string;
}
