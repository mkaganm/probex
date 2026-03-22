package models

import "time"

// TestResult holds the outcome of a single test case execution.
type TestResult struct {
	TestCaseID   string        `json:"test_case_id" yaml:"test_case_id"`
	TestName     string        `json:"test_name" yaml:"test_name"`
	Status       TestStatus    `json:"status" yaml:"status"`
	Category     TestCategory  `json:"category" yaml:"category"`
	Severity     Severity      `json:"severity" yaml:"severity"`
	Duration     time.Duration `json:"duration" yaml:"duration"`
	Request      TestRequest   `json:"request" yaml:"request"`
	Response     *TestResponse `json:"response,omitempty" yaml:"response,omitempty"`
	Assertions   []AssertionResult `json:"assertions" yaml:"assertions"`
	Error        string        `json:"error,omitempty" yaml:"error,omitempty"`
	ExecutedAt   time.Time     `json:"executed_at" yaml:"executed_at"`
}

// TestResponse captures the actual HTTP response.
type TestResponse struct {
	StatusCode  int               `json:"status_code" yaml:"status_code"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body        string            `json:"body,omitempty" yaml:"body,omitempty"`
	Duration    time.Duration     `json:"duration" yaml:"duration"`
}

// AssertionResult holds the outcome of a single assertion.
type AssertionResult struct {
	Assertion Assertion `json:"assertion" yaml:"assertion"`
	Passed    bool      `json:"passed" yaml:"passed"`
	Actual    any       `json:"actual,omitempty" yaml:"actual,omitempty"`
	Message   string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// TestStatus represents the status of a test execution.
type TestStatus string

const (
	StatusPassed  TestStatus = "passed"
	StatusFailed  TestStatus = "failed"
	StatusError   TestStatus = "error"
	StatusSkipped TestStatus = "skipped"
)

// RunSummary holds aggregate results for a test run.
type RunSummary struct {
	ProfileID   string        `json:"profile_id" yaml:"profile_id"`
	TotalTests  int           `json:"total_tests" yaml:"total_tests"`
	Passed      int           `json:"passed" yaml:"passed"`
	Failed      int           `json:"failed" yaml:"failed"`
	Errors      int           `json:"errors" yaml:"errors"`
	Skipped     int           `json:"skipped" yaml:"skipped"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	Results     []TestResult  `json:"results" yaml:"results"`
	StartedAt   time.Time     `json:"started_at" yaml:"started_at"`
	FinishedAt  time.Time     `json:"finished_at" yaml:"finished_at"`
	BySeverity  map[Severity]int  `json:"by_severity" yaml:"by_severity"`
	ByCategory  map[TestCategory]int `json:"by_category" yaml:"by_category"`
}
