package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Alert represents a notification about anomalies or drift.
type Alert struct {
	Timestamp time.Time       `json:"timestamp"`
	Endpoint  string          `json:"endpoint"`
	Anomalies []Anomaly       `json:"anomalies,omitempty"`
	Drifts    []Drift         `json:"drifts,omitempty"`
	Severity  models.Severity `json:"severity"`
	Message   string          `json:"message"`
}

// AlertTarget is a notification destination.
type AlertTarget interface {
	Send(ctx context.Context, alert Alert) error
	Type() string
}

// Alerter sends notifications when anomalies or drift are detected.
type Alerter struct {
	targets []AlertTarget
}

// NewAlerter creates a new Alerter with the given targets.
func NewAlerter(targets ...AlertTarget) *Alerter {
	return &Alerter{targets: targets}
}

// Send sends an alert to all configured targets.
func (a *Alerter) Send(ctx context.Context, alert Alert) error {
	var errs []string
	for _, t := range a.targets {
		if err := t.Send(ctx, alert); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", t.Type(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("alert errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// HasTargets returns true if the alerter has configured targets.
func (a *Alerter) HasTargets() bool {
	return len(a.targets) > 0
}

// ParseTargets parses a notification target string into AlertTarget instances.
// Format: "stdout", "webhook:https://...", "slack:https://hooks.slack.com/..."
// Multiple targets can be comma-separated.
func ParseTargets(notify string) []AlertTarget {
	if notify == "" {
		return []AlertTarget{&StdoutTarget{}}
	}

	var targets []AlertTarget
	for _, part := range strings.Split(notify, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "stdout" {
			targets = append(targets, &StdoutTarget{})
			continue
		}
		if strings.HasPrefix(part, "webhook:") {
			url := strings.TrimPrefix(part, "webhook:")
			targets = append(targets, &WebhookTarget{URL: url})
			continue
		}
		if strings.HasPrefix(part, "slack:") {
			url := strings.TrimPrefix(part, "slack:")
			targets = append(targets, &SlackTarget{WebhookURL: url})
			continue
		}
		// Default: treat as stdout
		targets = append(targets, &StdoutTarget{})
	}

	if len(targets) == 0 {
		targets = append(targets, &StdoutTarget{})
	}
	return targets
}

// --- StdoutTarget ---

// StdoutTarget prints alerts to the terminal.
type StdoutTarget struct{}

func (t *StdoutTarget) Type() string { return "stdout" }

func (t *StdoutTarget) Send(_ context.Context, alert Alert) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s %s\n", strings.ToUpper(string(alert.Severity)), alert.Endpoint, alert.Message))
	for _, a := range alert.Anomalies {
		sb.WriteString(fmt.Sprintf("  ANOMALY: %s\n", a.Description))
	}
	for _, d := range alert.Drifts {
		sb.WriteString(fmt.Sprintf("  DRIFT: %s %s (was: %s, now: %s)\n", d.Field, d.Change, d.OldValue, d.NewValue))
	}
	fmt.Print(sb.String())
	return nil
}

// --- WebhookTarget ---

// WebhookTarget sends alerts to a webhook URL as JSON POST.
type WebhookTarget struct {
	URL    string
	client *http.Client
}

func (t *WebhookTarget) Type() string { return "webhook" }

func (t *WebhookTarget) Send(ctx context.Context, alert Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	client := t.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// --- SlackTarget ---

// SlackTarget sends alerts to a Slack webhook.
type SlackTarget struct {
	WebhookURL string
	client     *http.Client
}

func (t *SlackTarget) Type() string { return "slack" }

func (t *SlackTarget) Send(ctx context.Context, alert Alert) error {
	// Build Slack message
	var sb strings.Builder
	icon := ":white_check_mark:"
	switch alert.Severity {
	case models.SeverityCritical:
		icon = ":rotating_light:"
	case models.SeverityHigh:
		icon = ":warning:"
	case models.SeverityMedium:
		icon = ":large_yellow_circle:"
	}

	sb.WriteString(fmt.Sprintf("%s *PROBEX Alert* — %s\n", icon, alert.Endpoint))
	sb.WriteString(fmt.Sprintf("*%s*\n", alert.Message))

	for _, a := range alert.Anomalies {
		sb.WriteString(fmt.Sprintf("• Anomaly: %s\n", a.Description))
	}
	for _, d := range alert.Drifts {
		sb.WriteString(fmt.Sprintf("• Drift: `%s` %s", d.Field, d.Change))
		if d.OldValue != "" {
			sb.WriteString(fmt.Sprintf(" (was: %s, now: %s)", d.OldValue, d.NewValue))
		}
		sb.WriteString("\n")
	}

	payload := map[string]string{"text": sb.String()}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := t.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
