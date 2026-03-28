package test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/report"
)

func testSummary() *models.RunSummary {
	return &models.RunSummary{
		TotalTests: 4,
		Passed:     2,
		Failed:     1,
		Errors:     1,
		Skipped:    0,
		Duration:   3 * time.Second,
		ProfileID:  "test-profile",
		Results: []models.TestResult{
			{
				TestName: "GET /users returns 200",
				Status:   models.StatusPassed,
				Category: models.CategoryHappyPath,
				Severity: models.SeverityMedium,
				Duration: 150 * time.Millisecond,
				Request:  models.TestRequest{Method: "GET", URL: "http://example.com/users"},
				Assertions: []models.AssertionResult{
					{Assertion: models.Assertion{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200}, Actual: 200, Passed: true},
				},
			},
			{
				TestName: "POST /users creates user",
				Status:   models.StatusPassed,
				Category: models.CategoryHappyPath,
				Severity: models.SeverityHigh,
				Duration: 200 * time.Millisecond,
				Request:  models.TestRequest{Method: "POST", URL: "http://example.com/users"},
			},
			{
				TestName: "SQL injection on /search",
				Status:   models.StatusFailed,
				Category: models.CategorySecurity,
				Severity: models.SeverityCritical,
				Duration: 100 * time.Millisecond,
				Request:  models.TestRequest{Method: "POST", URL: "http://example.com/search"},
				Assertions: []models.AssertionResult{
					{Assertion: models.Assertion{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 400}, Actual: 200, Passed: false, Message: "expected 400, got 200"},
				},
			},
			{
				TestName: "Connection timeout test",
				Status:   models.StatusError,
				Category: models.CategoryEdgeCase,
				Severity: models.SeverityLow,
				Duration: 5 * time.Second,
				Request:  models.TestRequest{Method: "GET", URL: "http://example.com/slow"},
				Error:    "connection timeout after 5s",
			},
		},
	}
}

func TestNewReporterJSON(t *testing.T) {
	r, err := report.NewReporter("json")
	if err != nil {
		t.Fatal(err)
	}
	if r.Format() != "json" {
		t.Errorf("expected json format, got %s", r.Format())
	}
}

func TestNewReporterJUnit(t *testing.T) {
	r, err := report.NewReporter("junit")
	if err != nil {
		t.Fatal(err)
	}
	if r.Format() != "junit" {
		t.Errorf("expected junit format, got %s", r.Format())
	}
}

func TestNewReporterHTML(t *testing.T) {
	r, err := report.NewReporter("html")
	if err != nil {
		t.Fatal(err)
	}
	if r.Format() != "html" {
		t.Errorf("expected html format, got %s", r.Format())
	}
}

func TestNewReporterInvalid(t *testing.T) {
	_, err := report.NewReporter("csv")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestJSONReporter(t *testing.T) {
	r := report.NewJSON()
	var buf bytes.Buffer
	if err := r.Generate(testSummary(), &buf); err != nil {
		t.Fatal(err)
	}

	var decoded models.RunSummary
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.TotalTests != 4 {
		t.Errorf("TotalTests: got %d, want 4", decoded.TotalTests)
	}
	if decoded.Passed != 2 {
		t.Errorf("Passed: got %d, want 2", decoded.Passed)
	}
}

func TestJUnitReporter(t *testing.T) {
	r := report.NewJUnit()
	var buf bytes.Buffer
	if err := r.Generate(testSummary(), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Verify it's valid XML.
	if !strings.HasPrefix(output, "<?xml") {
		t.Error("JUnit output should start with XML declaration")
	}

	// Parse the XML.
	type testSuites struct {
		XMLName  xml.Name `xml:"testsuites"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Errors   int      `xml:"errors,attr"`
	}
	var suites testSuites
	if err := xml.Unmarshal(buf.Bytes(), &suites); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	if suites.Tests != 4 {
		t.Errorf("Tests attr: got %d, want 4", suites.Tests)
	}
	if suites.Failures != 1 {
		t.Errorf("Failures attr: got %d, want 1", suites.Failures)
	}
	if suites.Errors != 1 {
		t.Errorf("Errors attr: got %d, want 1", suites.Errors)
	}
}

func TestHTMLReporter(t *testing.T) {
	r := report.NewHTML()
	var buf bytes.Buffer
	if err := r.Generate(testSummary(), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "<html") {
		t.Error("HTML output should contain <html")
	}
	if !strings.Contains(output, "PROBEX") {
		t.Error("HTML should contain PROBEX branding")
	}
}

func TestJSONReporterEmptySummary(t *testing.T) {
	r := report.NewJSON()
	var buf bytes.Buffer
	if err := r.Generate(&models.RunSummary{}, &buf); err != nil {
		t.Fatal(err)
	}

	var decoded models.RunSummary
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.TotalTests != 0 {
		t.Errorf("expected 0 tests, got %d", decoded.TotalTests)
	}
}

func TestJUnitReporterEmptySummary(t *testing.T) {
	r := report.NewJUnit()
	var buf bytes.Buffer
	if err := r.Generate(&models.RunSummary{}, &buf); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "testsuites") {
		t.Error("should still produce valid XML structure")
	}
}
