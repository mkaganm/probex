package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Fuzzer generates mutation-based fuzz test cases.
type Fuzzer struct{}

// NewFuzzer creates a new Fuzzer generator.
func NewFuzzer() *Fuzzer { return &Fuzzer{} }

// Category returns the test category.
func (f *Fuzzer) Category() models.TestCategory { return models.CategoryFuzz }

var specialCharPayloads = []struct {
	label string
	value string
}{
	{"null_bytes", "test\x00value"},
	{"tab_newline", "test\t\n\rvalue"},
	{"backslash", "test\\value\\\\end"},
	{"single_quotes", "test'value'end"},
	{"double_quotes", `test"value"end`},
	{"angle_brackets", "test<>value"},
	{"ampersand_semicolon", "test&value;end"},
	{"pipe_backtick", "test|value`end"},
	{"dollar_brace", "test${value}end"},
}

var unicodePayloads = []struct {
	label string
	value string
}{
	{"null_byte_unicode", "test\u0000value"},
	{"rtl_marker", "test\u200Fvalue\u200Eend"},
	{"zero_width_space", "test\u200Bvalue"},
	{"emoji", "test\U0001F4A9\U0001F600value"},
	{"zalgo", "t\u0354\u0353\u0354e\u0363\u0364\u0365s\u0346\u0347t"},
	{"bom", "\uFEFFtest"},
	{"replacement_char", "test\uFFFDvalue"},
	{"max_codepoint", "test\U0010FFFFvalue"},
}

var formatStringPayloads = []struct {
	label string
	value string
}{
	{"percent_s", "%s%s%s%s%s"},
	{"percent_x", "%x%x%x%x"},
	{"percent_n", "%n%n%n%n"},
	{"percent_d", "%d%d%d%d"},
	{"long_format", strings.Repeat("%s", 100)},
}

var typeConfusionPayloads = []struct {
	label   string
	forType string
	value   any
}{
	{"number_as_string", "string", 99999},
	{"bool_as_string", "string", true},
	{"float_as_int", "integer", 1.5},
	{"negative_float", "integer", -0.1},
	{"bool_as_int", "integer", false},
	{"string_true", "boolean", "true"},
	{"int_one_as_bool", "boolean", 1},
	{"empty_object", "string", map[string]any{}},
	{"nested_array", "string", []any{[]any{[]any{"deep"}}}},
}

// Generate creates fuzz test cases for an endpoint.
func (f *Fuzzer) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase

	method := strings.ToUpper(endpoint.Method)
	hasBody := method == "POST" || method == "PUT" || method == "PATCH"

	if !hasBody || endpoint.RequestBody == nil || endpoint.RequestBody.Properties == nil {
		return tests, nil
	}

	baseReq := buildBaseRequest(endpoint)
	exampleBody := buildExampleBody(endpoint.RequestBody)
	bodyMap, ok := exampleBody.(map[string]any)
	if !ok {
		return tests, nil
	}

	// Only server should not crash assertion - 500 is the main concern
	noCrashAssertions := []models.Assertion{
		{
			Type:     models.AssertStatusCode,
			Target:   "status_code",
			Operator: "ne",
			Expected: 500,
		},
	}

	for fieldName, prop := range endpoint.RequestBody.Properties {
		if prop.Type != "string" && prop.Type != "" {
			continue
		}

		// 1. Special characters
		for _, sp := range specialCharPayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = sp.value
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("Fuzz special_chars '%s' field '%s'", sp.label, fieldName),
				Description: fmt.Sprintf("Special character fuzz '%s' on field '%s'", sp.label, fieldName),
				Category:    models.CategoryFuzz,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: noCrashAssertions,
			})
		}

		// 2. Unicode edge cases
		for _, uc := range unicodePayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = uc.value
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("Fuzz unicode '%s' field '%s'", uc.label, fieldName),
				Description: fmt.Sprintf("Unicode fuzz '%s' on field '%s'", uc.label, fieldName),
				Category:    models.CategoryFuzz,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: noCrashAssertions,
			})
		}

		// 3. Format string payloads
		for _, fs := range formatStringPayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = fs.value
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("Fuzz format_string '%s' field '%s'", fs.label, fieldName),
				Description: fmt.Sprintf("Format string fuzz '%s' on field '%s'", fs.label, fieldName),
				Category:    models.CategoryFuzz,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: noCrashAssertions,
			})
		}
	}

	// 4. Type confusion - apply to all fields based on their type
	for fieldName, prop := range endpoint.RequestBody.Properties {
		for _, tc := range typeConfusionPayloads {
			if tc.forType != prop.Type {
				continue
			}
			mutated := copyMap(bodyMap)
			mutated[fieldName] = tc.value
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("Fuzz type_confusion '%s' field '%s'", tc.label, fieldName),
				Description: fmt.Sprintf("Type confusion '%s' on field '%s' (expected %s)", tc.label, fieldName, prop.Type),
				Category:    models.CategoryFuzz,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: noCrashAssertions,
			})
		}
	}

	return tests, nil
}
