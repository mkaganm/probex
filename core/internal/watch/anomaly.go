package watch

import (
	"fmt"
	"math"
	"time"

	"github.com/probex/probex/internal/models"
)

// AnomalyDetector detects anomalous API behavior using statistical methods.
type AnomalyDetector struct {
	threshold float64 // z-score threshold
}

// NewAnomalyDetector creates a new detector with a z-score threshold.
func NewAnomalyDetector(threshold float64) *AnomalyDetector {
	if threshold <= 0 {
		threshold = 2.0
	}
	return &AnomalyDetector{threshold: threshold}
}

// Anomaly represents a detected anomaly.
type Anomaly struct {
	EndpointID  string          `json:"endpoint_id"`
	Metric      string          `json:"metric"`
	Expected    float64         `json:"expected"`
	Actual      float64         `json:"actual"`
	ZScore      float64         `json:"z_score"`
	Severity    models.Severity `json:"severity"`
	Description string          `json:"description"`
}

// CheckResponseTime checks if the actual response time is anomalous
// compared to the baseline.
func (d *AnomalyDetector) CheckResponseTime(endpointID string, actual time.Duration, baseline *models.EndpointBaseline) *Anomaly {
	if baseline == nil || baseline.SampleCount < 2 {
		return nil
	}

	avg := float64(baseline.AvgResponseTime)
	if avg == 0 {
		return nil
	}

	// Estimate stddev from p50/p95 spread (p95 - p50 ≈ 1.65σ)
	p50 := float64(baseline.P50ResponseTime)
	p95 := float64(baseline.P95ResponseTime)
	stddev := (p95 - p50) / 1.65
	if stddev <= 0 {
		// Fallback: use 20% of avg as stddev estimate
		stddev = avg * 0.2
	}

	actualF := float64(actual)
	zScore := (actualF - avg) / stddev

	if math.Abs(zScore) < d.threshold {
		return nil
	}

	return &Anomaly{
		EndpointID:  endpointID,
		Metric:      "response_time",
		Expected:    avg,
		Actual:      actualF,
		ZScore:      zScore,
		Severity:    zScoreSeverity(math.Abs(zScore)),
		Description: fmt.Sprintf("Response time %.0fms deviates from avg %.0fms (z=%.2f)", actualF/float64(time.Millisecond), avg/float64(time.Millisecond), zScore),
	}
}

// CheckStatusCode checks if the actual status code is unexpected
// based on the baseline distribution.
func (d *AnomalyDetector) CheckStatusCode(endpointID string, actual int, baseline *models.EndpointBaseline) *Anomaly {
	if baseline == nil || len(baseline.StatusCodeDist) == 0 {
		return nil
	}

	// Check if this status code was ever seen in baseline
	if _, seen := baseline.StatusCodeDist[actual]; seen {
		return nil
	}

	// Determine severity based on the status code
	sev := models.SeverityMedium
	if actual >= 500 {
		sev = models.SeverityCritical
	} else if actual >= 400 {
		sev = models.SeverityHigh
	}

	// Find the most common status code for description
	mostCommon := 0
	maxCount := 0
	for code, count := range baseline.StatusCodeDist {
		if count > maxCount {
			mostCommon = code
			maxCount = count
		}
	}

	return &Anomaly{
		EndpointID:  endpointID,
		Metric:      "status_code",
		Expected:    float64(mostCommon),
		Actual:      float64(actual),
		ZScore:      0, // Not z-score based
		Severity:    sev,
		Description: fmt.Sprintf("Unexpected status code %d (baseline most common: %d)", actual, mostCommon),
	}
}

// zScoreSeverity maps z-score magnitude to severity.
func zScoreSeverity(absZ float64) models.Severity {
	switch {
	case absZ > 3.0:
		return models.SeverityCritical
	case absZ > 2.5:
		return models.SeverityHigh
	default:
		return models.SeverityMedium
	}
}
