package models

import "time"

// APIProfile is the complete learned profile of an API.
type APIProfile struct {
	ID          string      `json:"id" yaml:"id"`
	Name        string      `json:"name" yaml:"name"`
	BaseURL     string      `json:"base_url" yaml:"base_url"`
	Endpoints   []Endpoint  `json:"endpoints" yaml:"endpoints"`
	Auth        *AuthInfo   `json:"auth,omitempty" yaml:"auth,omitempty"`
	Baseline    *Baseline   `json:"baseline,omitempty" yaml:"baseline,omitempty"`
	CreatedAt   time.Time   `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" yaml:"updated_at"`
	ScanConfig  ScanConfig  `json:"scan_config" yaml:"scan_config"`
}

// Baseline holds performance baseline statistics for the API.
type Baseline struct {
	Endpoints map[string]*EndpointBaseline `json:"endpoints" yaml:"endpoints"`
}

// EndpointBaseline holds baseline stats for a single endpoint.
type EndpointBaseline struct {
	EndpointID       string        `json:"endpoint_id" yaml:"endpoint_id"`
	AvgResponseTime  time.Duration `json:"avg_response_time" yaml:"avg_response_time"`
	P50ResponseTime  time.Duration `json:"p50_response_time" yaml:"p50_response_time"`
	P95ResponseTime  time.Duration `json:"p95_response_time" yaml:"p95_response_time"`
	P99ResponseTime  time.Duration `json:"p99_response_time" yaml:"p99_response_time"`
	StatusCodeDist   map[int]int   `json:"status_code_dist" yaml:"status_code_dist"`
	SampleCount      int           `json:"sample_count" yaml:"sample_count"`
}

// ScanConfig holds the configuration used during scanning.
type ScanConfig struct {
	MaxDepth    int           `json:"max_depth" yaml:"max_depth"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	Concurrency int           `json:"concurrency" yaml:"concurrency"`
	AuthHeader  string        `json:"auth_header,omitempty" yaml:"auth_header,omitempty"`
}
