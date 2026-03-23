package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/report"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate test reports",
	Long: `Generate reports from the last test run in various formats.

Supported formats:
  - json:  Machine-readable JSON (for pipelines)
  - junit: JUnit XML (for CI tools like Jenkins, GitHub Actions)
  - html:  Human-readable HTML report

Examples:
  probex report --format json
  probex report --format junit -o results.xml
  probex report --format html -o report.html`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().String("format", "json", "report format (json, junit, html)")
	reportCmd.Flags().StringP("output", "o", "", "output file path (default: stdout)")
	reportCmd.Flags().String("run", "", "specific run ID to report on")

	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	runID, _ := cmd.Flags().GetString("run")

	// Create the reporter for the requested format.
	reporter, err := report.NewReporter(format)
	if err != nil {
		return err
	}

	// Load results from storage.
	summary, err := loadResults(runID)
	if err != nil {
		return fmt.Errorf("failed to load results: %w", err)
	}

	// Determine output destination.
	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	if err := reporter.Generate(summary, w); err != nil {
		return fmt.Errorf("failed to generate %s report: %w", format, err)
	}

	if output != "" {
		color.Green("Report written to %s (%s format)", output, format)
	}

	return nil
}

// loadResults loads a RunSummary from storage. If runID is provided, it loads
// results from the specific run file; otherwise it loads the latest results.
func loadResults(runID string) (*models.RunSummary, error) {
	store, err := storage.New("")
	if err != nil {
		return nil, err
	}

	if runID == "" {
		return store.LoadResults()
	}

	// Load a specific run by matching against available run files.
	runs, err := store.ListRuns()
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}

	for _, run := range runs {
		if run.Name == runID || run.Name == "results_"+runID+".json" {
			data, err := os.ReadFile(run.Path)
			if err != nil {
				return nil, err
			}
			var summary models.RunSummary
			if err := json.Unmarshal(data, &summary); err != nil {
				return nil, err
			}
			return &summary, nil
		}
	}

	return nil, fmt.Errorf("run %q not found; use 'probex report' without --run for the latest", runID)
}
