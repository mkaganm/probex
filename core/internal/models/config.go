package models

import "time"

// Config is the top-level probex configuration.
type Config struct {
	Version string       `json:"version" yaml:"version"`
	Target  TargetConfig `json:"target" yaml:"target"`
	Scan    ScanOptions  `json:"scan" yaml:"scan"`
	Run     RunOptions   `json:"run" yaml:"run"`
	Watch   WatchOptions `json:"watch" yaml:"watch"`
	Guard   GuardOptions `json:"guard" yaml:"guard"`
	AI      AIConfig     `json:"ai" yaml:"ai"`
	Report  ReportConfig `json:"report" yaml:"report"`
}

// TargetConfig defines the API target.
type TargetConfig struct {
	BaseURL    string            `json:"base_url" yaml:"base_url"`
	AuthHeader string            `json:"auth_header,omitempty" yaml:"auth_header,omitempty"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ScanOptions configures the scan behavior.
type ScanOptions struct {
	MaxDepth    int           `json:"max_depth" yaml:"max_depth"`
	Concurrency int           `json:"concurrency" yaml:"concurrency"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	Wordlist    string        `json:"wordlist,omitempty" yaml:"wordlist,omitempty"`
	FollowLinks bool          `json:"follow_links" yaml:"follow_links"`
}

// RunOptions configures the test run behavior.
type RunOptions struct {
	Concurrency int            `json:"concurrency" yaml:"concurrency"`
	Timeout     time.Duration  `json:"timeout" yaml:"timeout"`
	Categories  []TestCategory `json:"categories,omitempty" yaml:"categories,omitempty"`
	UseAI       bool           `json:"use_ai" yaml:"use_ai"`
	StopOnFail  bool           `json:"stop_on_fail" yaml:"stop_on_fail"`
}

// WatchOptions configures the watch mode.
type WatchOptions struct {
	Interval      time.Duration `json:"interval" yaml:"interval"`
	Endpoints     []string      `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	NotifySlack   string        `json:"notify_slack,omitempty" yaml:"notify_slack,omitempty"`
	NotifyWebhook string        `json:"notify_webhook,omitempty" yaml:"notify_webhook,omitempty"`
}

// GuardOptions configures the CI guard mode.
type GuardOptions struct {
	FailOn     []Severity `json:"fail_on" yaml:"fail_on"`
	ReportFile string     `json:"report_file,omitempty" yaml:"report_file,omitempty"`
}

// AIConfig configures the AI integration.
type AIConfig struct {
	Mode   string   `json:"mode" yaml:"mode"` // local, cloud, hybrid, offline
	Local  LocalAI  `json:"local" yaml:"local"`
	Cloud  CloudAI  `json:"cloud" yaml:"cloud"`
	Budget AIBudget `json:"budget" yaml:"budget"`
}

// LocalAI configures the local AI provider.
type LocalAI struct {
	Provider string `json:"provider" yaml:"provider"` // ollama, llamacpp
	Model    string `json:"model" yaml:"model"`
}

// CloudAI configures the cloud AI provider.
type CloudAI struct {
	Provider  string   `json:"provider" yaml:"provider"` // anthropic, openai
	Model     string   `json:"model" yaml:"model"`
	APIKey    string   `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	UseFor    []string `json:"use_for,omitempty" yaml:"use_for,omitempty"`
	NeverSend []string `json:"never_send,omitempty" yaml:"never_send,omitempty"`
}

// AIBudget controls AI spending.
type AIBudget struct {
	MaxMonthlyCost          float64 `json:"max_monthly_cost" yaml:"max_monthly_cost"`
	PreferLocalWhenPossible bool    `json:"prefer_local_when_possible" yaml:"prefer_local_when_possible"`
}

// ReportConfig configures report output.
type ReportConfig struct {
	Format string `json:"format" yaml:"format"` // json, junit, html
	Output string `json:"output" yaml:"output"` // file path or "stdout"
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Version: "1",
		Scan: ScanOptions{
			MaxDepth:    3,
			Concurrency: 10,
			Timeout:     30 * time.Second,
			FollowLinks: true,
		},
		Run: RunOptions{
			Concurrency: 5,
			Timeout:     30 * time.Second,
		},
		Watch: WatchOptions{
			Interval: 5 * time.Minute,
		},
		Guard: GuardOptions{
			FailOn: []Severity{SeverityCritical, SeverityHigh},
		},
		AI: AIConfig{
			Mode: "offline",
			Local: LocalAI{
				Provider: "ollama",
				Model:    "qwen3:4b",
			},
			Cloud: CloudAI{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-6",
			},
			Budget: AIBudget{
				MaxMonthlyCost:          20,
				PreferLocalWhenPossible: true,
			},
		},
		Report: ReportConfig{
			Format: "json",
			Output: "stdout",
		},
	}
}
