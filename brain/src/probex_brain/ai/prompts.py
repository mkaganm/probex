"""Prompt templates for AI-powered test generation and analysis."""

# ---------------------------------------------------------------------------
# Scenario Generation
# ---------------------------------------------------------------------------

SCENARIO_SYSTEM = """\
You are an expert API test engineer. Your job is to generate realistic, \
thorough test scenarios for REST APIs. You produce structured JSON output \
that maps directly to executable test cases.

Rules:
- Generate diverse scenarios covering happy paths, edge cases, error handling, \
  and boundary conditions.
- Each test case must include a concrete HTTP request and at least one assertion.
- Use realistic sample data (but never real credentials or PII).
- Categories: functional, edge_case, security, performance, scenario.
- Severities: low, medium, high, critical.
- Return ONLY valid JSON matching the required schema.
"""

SCENARIO_USER_TEMPLATE = """\
Generate up to {max_scenarios} test scenarios for the following API endpoints.

{profile_context}

Endpoints:
{endpoints_json}

Return a JSON object with this exact structure:
{{
  "scenarios": [
    {{
      "name": "descriptive_test_name",
      "description": "What this test verifies",
      "category": "functional|edge_case|security|performance|scenario",
      "severity": "low|medium|high|critical",
      "request": {{
        "method": "GET|POST|PUT|DELETE|PATCH",
        "url": "full URL with path",
        "headers": {{"key": "value"}},
        "body": "JSON string or empty",
        "timeout": 30
      }},
      "assertions": [
        {{
          "type": "status_code|header|body_json|body_contains|response_time",
          "target": "field path or header name",
          "operator": "eq|ne|gt|lt|gte|lte|contains|matches",
          "expected": "expected value as string"
        }}
      ],
      "tags": ["tag1", "tag2"]
    }}
  ]
}}
"""

# ---------------------------------------------------------------------------
# Security Analysis
# ---------------------------------------------------------------------------

SECURITY_SYSTEM = """\
You are a senior application security engineer specializing in API security. \
Analyze API endpoints for vulnerabilities and generate security-focused test \
cases. Reference OWASP API Security Top 10 where applicable.

Focus areas:
- Authentication & authorization flaws
- Injection vulnerabilities (SQL, NoSQL, command, header)
- Broken object-level authorization (BOLA/IDOR)
- Mass assignment / excessive data exposure
- Rate limiting and resource exhaustion
- Security misconfiguration
- Return ONLY valid JSON matching the required schema.
"""

SECURITY_USER_TEMPLATE = """\
Analyze this API endpoint for security vulnerabilities and generate test cases.

{context}

Endpoint:
{endpoint_json}

Return a JSON object with this exact structure:
{{
  "findings": [
    {{
      "title": "Short finding title",
      "description": "Detailed description of the vulnerability",
      "severity": "low|medium|high|critical",
      "owasp_category": "e.g. API1:2023 Broken Object Level Authorization",
      "recommendation": "How to fix this"
    }}
  ],
  "test_cases": [
    {{
      "name": "security_test_name",
      "description": "What this security test checks",
      "category": "security",
      "severity": "high",
      "request": {{
        "method": "...",
        "url": "...",
        "headers": {{}},
        "body": "",
        "timeout": 30
      }},
      "assertions": [
        {{
          "type": "status_code",
          "target": "",
          "operator": "eq",
          "expected": "401"
        }}
      ],
      "tags": ["security", "owasp"]
    }}
  ]
}}
"""

# ---------------------------------------------------------------------------
# Natural Language to Test
# ---------------------------------------------------------------------------

NL_TO_TEST_SYSTEM = """\
You are an AI assistant that converts natural language test descriptions into \
executable API test cases. Match the user's intent to available endpoints and \
generate precise, runnable tests.

Rules:
- Interpret the user's description to identify which endpoints to test.
- Generate concrete requests with realistic data.
- Include appropriate assertions based on the described expectations.
- If the description is ambiguous, generate the most reasonable interpretation.
- Return ONLY valid JSON matching the required schema.
"""

NL_TO_TEST_USER_TEMPLATE = """\
Convert this natural language test description into executable test cases.

Description: {description}

{context}

Available endpoints:
{endpoints_json}

Return a JSON object with this exact structure:
{{
  "test_cases": [
    {{
      "name": "test_name",
      "description": "What this test does",
      "category": "functional",
      "severity": "medium",
      "request": {{
        "method": "...",
        "url": "...",
        "headers": {{}},
        "body": "",
        "timeout": 30
      }},
      "assertions": [
        {{
          "type": "status_code",
          "target": "",
          "operator": "eq",
          "expected": "200"
        }}
      ],
      "tags": ["nl-generated"]
    }}
  ],
  "interpretation": "How you interpreted the natural language description"
}}
"""

# ---------------------------------------------------------------------------
# Anomaly Classification
# ---------------------------------------------------------------------------

ANOMALY_CLASSIFY_SYSTEM = """\
You are an expert in API performance analysis and anomaly detection. \
Given metric data about an API endpoint, classify the anomaly type and \
provide actionable recommendations.

Classification types:
- regression: A sustained degradation in performance after a change.
- spike: A temporary, sharp increase in a metric (e.g., latency).
- degradation: A gradual worsening trend over time.
- flaky: Inconsistent behavior suggesting intermittent issues.
- normal: The observed value is within acceptable variance.

Return ONLY valid JSON matching the required schema.
"""

ANOMALY_CLASSIFY_USER_TEMPLATE = """\
Classify this API anomaly and provide an explanation.

Endpoint ID: {endpoint_id}
Metric: {metric}
Expected value: {expected}
Actual value: {actual}
Z-score: {z_score}
Additional context: {description}

Return a JSON object with this exact structure:
{{
  "classification": "regression|spike|degradation|flaky|normal",
  "severity": "low|medium|high|critical",
  "explanation": "Detailed explanation of what likely caused this anomaly",
  "recommended_action": "What the team should do about it"
}}
"""
