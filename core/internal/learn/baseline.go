package learn

import (
	"sort"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// BuildBaseline calculates performance baselines from grouped HAR entries.
func BuildBaseline(grouped map[EndpointKey][]Entry) *models.Baseline {
	baseline := &models.Baseline{
		Endpoints: make(map[string]*models.EndpointBaseline),
	}

	for key, entries := range grouped {
		eb := buildEndpointBaseline(key, entries)
		baseline.Endpoints[key.String()] = eb
	}

	return baseline
}

// buildEndpointBaseline calculates baseline statistics for a single endpoint.
func buildEndpointBaseline(key EndpointKey, entries []Entry) *models.EndpointBaseline {
	eb := &models.EndpointBaseline{
		EndpointID:     endpointID(key),
		SampleCount:    len(entries),
		StatusCodeDist: make(map[int]int),
	}

	// Collect response times in milliseconds.
	var times []float64
	for _, e := range entries {
		// Prefer the total time from timings; fall back to entry.Time.
		ms := e.Timings.TotalMillis()
		if ms <= 0 {
			ms = e.Time
		}
		if ms > 0 {
			times = append(times, ms)
		}

		// Status code distribution.
		eb.StatusCodeDist[e.Response.Status]++
	}

	if len(times) == 0 {
		return eb
	}

	// Sort for percentile calculations.
	sort.Float64s(times)

	eb.AvgResponseTime = millisToDuration(avg(times))
	eb.P50ResponseTime = millisToDuration(percentile(times, 50))
	eb.P95ResponseTime = millisToDuration(percentile(times, 95))
	eb.P99ResponseTime = millisToDuration(percentile(times, 99))

	return eb
}

// avg returns the arithmetic mean of a slice of float64 values.
func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// percentile returns the p-th percentile from a sorted slice using nearest-rank.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	// Nearest-rank method.
	rank := (p / 100) * float64(len(sorted))
	idx := int(rank)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

// millisToDuration converts milliseconds to a time.Duration.
func millisToDuration(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}
