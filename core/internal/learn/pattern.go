package learn

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
)

// Pattern regexes for field-level format detection.
var (
	patEmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	patUUIDRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	patDateTimeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	patDateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	patURLRegex      = regexp.MustCompile(`^https?://`)
)

// PatternReport holds detected field patterns per endpoint.
type PatternReport struct {
	Endpoints map[string][]FieldPattern `json:"endpoints"`
}

// FieldPattern describes a detected pattern for a response body field.
type FieldPattern struct {
	// FieldPath is the dot-separated path to the field (e.g., "user.email").
	FieldPath string `json:"field_path"`
	// Format is the detected format: email, uuid, date, datetime, url, numeric_id, enum.
	Format string `json:"format"`
	// SampleValues contains a few example values.
	SampleValues []string `json:"sample_values,omitempty"`
	// EnumValues is populated when Format is "enum".
	EnumValues []string `json:"enum_values,omitempty"`
	// Confidence is a rough confidence score from 0.0 to 1.0.
	Confidence float64 `json:"confidence"`
}

// maxEnumDistinct is the maximum number of distinct values to consider a field an enum.
const maxEnumDistinct = 10

// MinePatterns analyzes response bodies across HAR entries to detect field-level patterns.
func MinePatterns(grouped map[EndpointKey][]Entry) *PatternReport {
	report := &PatternReport{
		Endpoints: make(map[string][]FieldPattern),
	}

	for key, entries := range grouped {
		patterns := analyzeEndpointPatterns(entries)
		if len(patterns) > 0 {
			report.Endpoints[key.String()] = patterns
		}
	}

	return report
}

// analyzeEndpointPatterns analyzes response bodies for a single endpoint group.
func analyzeEndpointPatterns(entries []Entry) []FieldPattern {
	// Collect field values across all response bodies.
	fieldValues := make(map[string][]string)

	for _, e := range entries {
		body := e.Response.Content.Text
		if body == "" {
			continue
		}

		var parsed any
		if err := json.Unmarshal([]byte(body), &parsed); err != nil {
			continue
		}

		collectFieldValues("", parsed, fieldValues)
	}

	// Analyze each field's values for patterns.
	var patterns []FieldPattern
	for fieldPath, values := range fieldValues {
		if p := detectFieldPattern(fieldPath, values); p != nil {
			patterns = append(patterns, *p)
		}
	}

	// Sort by field path for deterministic output.
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].FieldPath < patterns[j].FieldPath
	})

	return patterns
}

// collectFieldValues recursively walks a parsed JSON value and collects string representations
// of leaf values keyed by their dot-separated path.
func collectFieldValues(prefix string, value any, out map[string][]string) {
	switch v := value.(type) {
	case map[string]any:
		for k, child := range v {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			collectFieldValues(path, child, out)
		}
	case []any:
		// For arrays, analyze the first few elements to find patterns.
		limit := len(v)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			collectFieldValues(prefix+"[]", v[i], out)
		}
	case string:
		if prefix != "" {
			out[prefix] = append(out[prefix], v)
		}
	case float64:
		if prefix != "" {
			out[prefix] = append(out[prefix], strconv.FormatFloat(v, 'f', -1, 64))
		}
	case bool:
		if prefix != "" {
			out[prefix] = append(out[prefix], strconv.FormatBool(v))
		}
	}
}

// detectFieldPattern analyzes a set of values for a field and returns a detected pattern if any.
func detectFieldPattern(fieldPath string, values []string) *FieldPattern {
	if len(values) == 0 {
		return nil
	}

	// Count how many values match each format.
	var emailCount, uuidCount, dateTimeCount, dateCount, urlCount, numericIDCount int
	for _, v := range values {
		switch {
		case patUUIDRegex.MatchString(v):
			uuidCount++
		case patEmailRegex.MatchString(v):
			emailCount++
		case patDateTimeRegex.MatchString(v):
			dateTimeCount++
		case patDateRegex.MatchString(v):
			dateCount++
		case patURLRegex.MatchString(v):
			urlCount++
		case isNumericID(v):
			numericIDCount++
		}
	}

	total := len(values)
	threshold := 0.8 // 80% of values must match for a format to be detected.

	// Check formats in priority order.
	type candidate struct {
		format string
		count  int
	}
	candidates := []candidate{
		{"uuid", uuidCount},
		{"email", emailCount},
		{"datetime", dateTimeCount},
		{"date", dateCount},
		{"url", urlCount},
		{"numeric_id", numericIDCount},
	}

	for _, c := range candidates {
		confidence := float64(c.count) / float64(total)
		if confidence >= threshold {
			return &FieldPattern{
				FieldPath:    fieldPath,
				Format:       c.format,
				SampleValues: sampleValues(values, 3),
				Confidence:   confidence,
			}
		}
	}

	// Check for enum pattern: small number of distinct values.
	if total >= 2 {
		distinct := distinctValues(values)
		if len(distinct) <= maxEnumDistinct && len(distinct) < total {
			return &FieldPattern{
				FieldPath:  fieldPath,
				Format:     "enum",
				EnumValues: distinct,
				Confidence: 1.0,
			}
		}
	}

	return nil
}

// isNumericID checks if a string looks like a numeric ID (positive integer, not too long).
func isNumericID(s string) bool {
	if len(s) == 0 || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// sampleValues returns up to n values from the slice.
func sampleValues(values []string, n int) []string {
	if len(values) <= n {
		result := make([]string, len(values))
		copy(result, values)
		return result
	}
	result := make([]string, n)
	copy(result, values[:n])
	return result
}

// distinctValues returns sorted unique values from the slice.
func distinctValues(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
