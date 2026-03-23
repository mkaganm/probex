package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mkaganm/probex/internal/models"
)

var (
	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D4FF")).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)

// Banner prints the probex banner with gradient colors and tagline.
func Banner() string {
	lines := []string{
		" ██████╗ ██████╗  ██████╗ ██████╗ ███████╗██╗  ██╗",
		" ██╔══██╗██╔══██╗██╔═══██╗██╔══██╗██╔════╝╚██╗██╔╝",
		" ██████╔╝██████╔╝██║   ██║██████╔╝█████╗   ╚███╔╝ ",
		" ██╔═══╝ ██╔══██╗██║   ██║██╔══██╗██╔══╝   ██╔██╗ ",
		" ██║     ██║  ██║╚██████╔╝██████╔╝███████╗██╔╝ ██╗",
		" ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝",
	}

	// Gradient from cyan to green.
	colors := []string{"#00D4FF", "#00DDEE", "#00E6CC", "#00EFAA", "#00F888", "#00FF88"}

	var result string
	for i, line := range lines {
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colors[i]))
		result += style.Render(line) + "\n"
	}

	tagline := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true).
		Render("  Zero-Test API Intelligence Engine  v1.0.0")

	return result + tagline
}

// Success prints a success message.
func Success(msg string) {
	fmt.Println(SuccessStyle.Render("✓ " + msg))
}

// Warning prints a warning message.
func Warning(msg string) {
	fmt.Println(WarningStyle.Render("⚠ " + msg))
}

// Error prints an error message.
func Error(msg string) {
	fmt.Println(ErrorStyle.Render("✗ " + msg))
}

// Info prints a dim info message.
func Info(msg string) {
	fmt.Println(DimStyle.Render("  " + msg))
}

// ScanSummary prints a summary of scan results.
func ScanSummary(endpoints int, duration time.Duration) {
	fmt.Println()
	fmt.Println(TitleStyle.Render("  Scan Complete"))
	Success(fmt.Sprintf("%d endpoints discovered in %s", endpoints, duration.Round(time.Millisecond)))
}

// EndpointList prints discovered endpoints in a table.
func EndpointList(endpoints []models.Endpoint) {
	t := NewTable("METHOD", "PATH", "AUTH", "SOURCE")
	for _, ep := range endpoints {
		authStr := "none"
		if ep.Auth != nil && ep.Auth.Type != models.AuthNone {
			authStr = string(ep.Auth.Type)
		}
		t.AddRow(ep.Method, ep.Path, authStr, string(ep.Source))
	}
	fmt.Println(t.Render())
}

// RunSummary prints test run results with colors.
func RunSummary(summary *models.RunSummary) {
	fmt.Println(TitleStyle.Render("  Test Results"))
	fmt.Println()

	// Overall stats
	if summary.Passed > 0 {
		Success(fmt.Sprintf("Passed:  %d", summary.Passed))
	}
	if summary.Failed > 0 {
		Error(fmt.Sprintf("Failed:  %d", summary.Failed))
	}
	if summary.Errors > 0 {
		Error(fmt.Sprintf("Errors:  %d", summary.Errors))
	}
	if summary.Skipped > 0 {
		Warning(fmt.Sprintf("Skipped: %d", summary.Skipped))
	}
	Info(fmt.Sprintf("Total:   %d in %s", summary.TotalTests, summary.Duration.Round(time.Millisecond)))
	fmt.Println()

	// Severity breakdown
	if len(summary.BySeverity) > 0 {
		fmt.Println("  By Severity:")
		for _, sev := range []models.Severity{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow, models.SeverityInfo} {
			if count, ok := summary.BySeverity[sev]; ok && count > 0 {
				style := DimStyle
				switch sev {
				case models.SeverityCritical:
					style = ErrorStyle
				case models.SeverityHigh:
					style = ErrorStyle
				case models.SeverityMedium:
					style = WarningStyle
				}
				fmt.Println(style.Render(fmt.Sprintf("    %-10s %d", sev, count)))
			}
		}
		fmt.Println()
	}

	// Category breakdown
	if len(summary.ByCategory) > 0 {
		fmt.Println("  By Category:")
		for cat, count := range summary.ByCategory {
			Info(fmt.Sprintf("%-15s %d", cat, count))
		}
		fmt.Println()
	}

	// Failed test details
	for _, r := range summary.Results {
		if r.Status == models.StatusFailed || r.Status == models.StatusError {
			Error(fmt.Sprintf("%s: %s", r.TestName, r.Error))
		}
	}
}
