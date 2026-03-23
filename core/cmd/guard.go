package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/report"
	"github.com/mkaganm/probex/internal/runner"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/mkaganm/probex/internal/ui"
	"github.com/spf13/cobra"
)

var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "CI/CD gate — fail on findings",
	Long: `Guard mode runs tests and exits with a non-zero code if issues are found.
Designed for CI/CD pipeline integration.

Examples:
  probex guard --ci
  probex guard --fail-on critical,high
  probex guard --report junit:results.xml
  probex guard --fail-on critical,high,medium --report html:report.html`,
	RunE: runGuard,
}

func init() {
	guardCmd.Flags().Bool("ci", false, "CI mode (non-interactive, exit code based)")
	guardCmd.Flags().StringSlice("fail-on", []string{"critical", "high"}, "severity levels that cause failure")
	guardCmd.Flags().String("report", "", "report output (format:path, e.g. junit:results.xml)")

	rootCmd.AddCommand(guardCmd)
}

func runGuard(cmd *cobra.Command, args []string) error {
	ciMode, _ := cmd.Flags().GetBool("ci")
	failOnStrs, _ := cmd.Flags().GetStringSlice("fail-on")
	reportSpec, _ := cmd.Flags().GetString("report")

	if !ciMode {
		fmt.Println(color.New(color.Bold).Sprint("Running guard checks..."))
		fmt.Println()
	}

	// Build fail-on severity set
	failOn := make(map[models.Severity]bool)
	for _, s := range failOnStrs {
		failOn[models.Severity(strings.TrimSpace(s))] = true
	}

	// Load profile
	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	if !store.ProfileExists() {
		return fmt.Errorf("no profile found — run 'probex scan' first")
	}
	profile, err := store.LoadProfile()
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}

	// Generate tests
	eng := generator.New(profile)
	tests, err := eng.Generate()
	if err != nil {
		return fmt.Errorf("generate tests: %w", err)
	}

	if !ciMode {
		ui.Info(fmt.Sprintf("Generated %d tests for %d endpoints", len(tests), len(profile.Endpoints)))
	}

	// Execute tests
	opts := models.RunOptions{
		Concurrency: 5,
		Timeout:     30 * time.Second,
	}
	exec := runner.New(opts)
	summary, err := exec.Execute(context.Background(), tests)
	if err != nil {
		return fmt.Errorf("execute tests: %w", err)
	}
	summary.ProfileID = profile.ID

	// Save results
	if err := store.SaveResults(summary); err != nil {
		if !ciMode {
			ui.Warning(fmt.Sprintf("Failed to save results: %v", err))
		}
	}

	// Write report if requested
	if reportSpec != "" {
		if err := writeGuardReport(reportSpec, summary); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	// Print summary
	if !ciMode {
		fmt.Println()
		ui.RunSummary(summary)
		fmt.Println()
	}

	// Check for failures matching fail-on severities
	failCount := 0
	for _, r := range summary.Results {
		if r.Status == models.StatusFailed || r.Status == models.StatusError {
			if failOn[r.Severity] {
				failCount++
			}
		}
	}

	if failCount > 0 {
		msg := fmt.Sprintf("Guard FAILED: %d findings at severity %v", failCount, failOnStrs)
		if ciMode {
			fmt.Fprintln(os.Stderr, msg)
		} else {
			ui.Error(msg)
		}
		os.Exit(1)
	}

	if !ciMode {
		ui.Success("Guard PASSED — no findings at specified severity levels")
	}
	return nil
}

// writeGuardReport parses "format:path" and writes the report.
func writeGuardReport(spec string, summary *models.RunSummary) error {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid report spec %q — use format:path (e.g. junit:results.xml)", spec)
	}
	format, path := parts[0], parts[1]

	reporter, err := report.NewReporter(format)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return reporter.Generate(summary, f)
}
