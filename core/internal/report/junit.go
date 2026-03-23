package report

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/mkaganm/probex/internal/models"
)

// JUnit XML structures for encoding.

type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Skipped    int              `xml:"skipped,attr"`
	Time       float64          `xml:"time,attr"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// JUnitReporter generates JUnit XML reports for CI/CD integration.
type JUnitReporter struct{}

// NewJUnit creates a new JUnitReporter.
func NewJUnit() *JUnitReporter { return &JUnitReporter{} }

// Format returns the reporter's format name.
func (r *JUnitReporter) Format() string { return "junit" }

// Generate writes a JUnit XML report to the writer.
func (r *JUnitReporter) Generate(summary *models.RunSummary, w io.Writer) error {
	// Group results by category to form test suites.
	suiteMap := make(map[models.TestCategory][]models.TestResult)
	for _, result := range summary.Results {
		suiteMap[result.Category] = append(suiteMap[result.Category], result)
	}

	var suites []junitTestSuite
	for category, results := range suiteMap {
		suite := junitTestSuite{
			Name: string(category),
		}
		for _, result := range results {
			tc := junitTestCase{
				Name:      result.TestName,
				ClassName: result.Request.Method + " " + result.Request.URL,
				Time:      result.Duration.Seconds(),
			}

			switch result.Status {
			case models.StatusFailed:
				suite.Failures++
				msg, body := buildFailureDetails(result)
				tc.Failure = &junitFailure{
					Message: msg,
					Type:    "AssertionFailure",
					Body:    body,
				}
			case models.StatusError:
				suite.Errors++
				tc.Error = &junitError{
					Message: result.Error,
					Type:    "TestError",
					Body:    result.Error,
				}
			case models.StatusSkipped:
				suite.Skipped++
				tc.Skipped = &junitSkipped{Message: "Test skipped"}
			}

			suite.Tests++
			suite.Time += result.Duration.Seconds()
			suite.TestCases = append(suite.TestCases, tc)
		}
		suites = append(suites, suite)
	}

	root := junitTestSuites{
		Tests:      summary.TotalTests,
		Failures:   summary.Failed,
		Errors:     summary.Errors,
		Skipped:    summary.Skipped,
		Time:       summary.Duration.Seconds(),
		TestSuites: suites,
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return err
	}
	// Trailing newline for cleanliness.
	_, err := io.WriteString(w, "\n")
	return err
}

// buildFailureDetails extracts a summary message and detailed body from failed assertions.
func buildFailureDetails(result models.TestResult) (message string, body string) {
	var failedAssertions []string
	for _, ar := range result.Assertions {
		if !ar.Passed {
			detail := fmt.Sprintf("[%s] %s %s expected=%v actual=%v",
				ar.Assertion.Type, ar.Assertion.Target, ar.Assertion.Operator,
				ar.Assertion.Expected, ar.Actual)
			if ar.Message != "" {
				detail += " — " + ar.Message
			}
			failedAssertions = append(failedAssertions, detail)
		}
	}
	if len(failedAssertions) == 0 {
		if result.Error != "" {
			return result.Error, result.Error
		}
		return "Test failed", "Test failed with no assertion details"
	}
	message = failedAssertions[0]
	body = strings.Join(failedAssertions, "\n")
	return message, body
}
