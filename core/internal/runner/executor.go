package runner

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Executor runs test cases concurrently and collects results.
type Executor struct {
	opts     models.RunOptions
	client   *http.Client
	asserter *AssertionEngine
}

// New creates a new test Executor.
func New(opts models.RunOptions) *Executor {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Executor{
		opts:     opts,
		client:   &http.Client{Timeout: timeout},
		asserter: NewAssertionEngine(),
	}
}

// Execute runs all provided test cases and returns a summary.
func (e *Executor) Execute(ctx context.Context, tests []models.TestCase) (*models.RunSummary, error) {
	start := time.Now()

	concurrency := e.opts.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var results []models.TestResult
	var wg sync.WaitGroup

	stopCh := make(chan struct{})
	var stopped atomic.Bool

	for _, tc := range tests {
		select {
		case <-ctx.Done():
			break
		case <-stopCh:
			break
		default:
		}

		if stopped.Load() {
			break
		}

		wg.Add(1)
		go func(test models.TestCase) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := e.executeOne(ctx, test)

			mu.Lock()
			results = append(results, result)
			if e.opts.StopOnFail && (result.Status == models.StatusFailed || result.Status == models.StatusError) {
				if stopped.CompareAndSwap(false, true) {
					close(stopCh)
				}
			}
			mu.Unlock()
		}(tc)
	}

	wg.Wait()

	summary := buildSummary(results, start)
	return summary, nil
}

func (e *Executor) executeOne(ctx context.Context, tc models.TestCase) models.TestResult {
	result := models.TestResult{
		TestCaseID: tc.ID,
		TestName:   tc.Name,
		Category:   tc.Category,
		Severity:   tc.Severity,
		Request:    tc.Request,
		ExecutedAt: time.Now(),
	}

	// Build HTTP request
	var bodyReader io.Reader
	if tc.Request.Body != "" {
		bodyReader = strings.NewReader(tc.Request.Body)
	}

	req, err := http.NewRequestWithContext(ctx, tc.Request.Method, tc.Request.URL, bodyReader)
	if err != nil {
		result.Status = models.StatusError
		result.Error = err.Error()
		return result
	}

	for k, v := range tc.Request.Headers {
		// Skip placeholder auth tokens
		if v == "{{auth_token}}" {
			continue
		}
		req.Header.Set(k, v)
	}

	// Execute request
	start := time.Now()
	resp, err := e.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		result.Status = models.StatusError
		result.Error = err.Error()
		result.Duration = duration
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		result.Status = models.StatusError
		result.Error = err.Error()
		result.Duration = duration
		return result
	}

	// Build test response
	testResp := &models.TestResponse{
		StatusCode: resp.StatusCode,
		Headers:    extractHeaders(resp),
		Body:       string(body),
		Duration:   duration,
	}
	result.Response = testResp
	result.Duration = duration

	// Run assertions
	allPassed := true
	for _, assertion := range tc.Assertions {
		ar := e.asserter.Evaluate(assertion, testResp)
		result.Assertions = append(result.Assertions, ar)
		if !ar.Passed {
			allPassed = false
			if result.Error == "" {
				result.Error = ar.Message
			}
		}
	}

	if allPassed {
		result.Status = models.StatusPassed
	} else {
		result.Status = models.StatusFailed
	}

	return result
}

func extractHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	return headers
}

func buildSummary(results []models.TestResult, start time.Time) *models.RunSummary {
	now := time.Now()
	summary := &models.RunSummary{
		TotalTests: len(results),
		Results:    results,
		StartedAt:  start,
		FinishedAt: now,
		Duration:   now.Sub(start),
		BySeverity: make(map[models.Severity]int),
		ByCategory: make(map[models.TestCategory]int),
	}

	for _, r := range results {
		switch r.Status {
		case models.StatusPassed:
			summary.Passed++
		case models.StatusFailed:
			summary.Failed++
		case models.StatusError:
			summary.Errors++
		case models.StatusSkipped:
			summary.Skipped++
		}
		summary.BySeverity[r.Severity]++
		summary.ByCategory[r.Category]++
	}

	return summary
}
