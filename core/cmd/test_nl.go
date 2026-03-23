package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/ai"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/runner"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/mkaganm/probex/internal/ui"
	"github.com/spf13/cobra"
)

var testNLCmd = &cobra.Command{
	Use:   "test [description]",
	Short: "Generate tests from natural language",
	Long: `Describe what you want to test in plain language and PROBEX generates
executable API tests using AI.

Requires the Python AI brain to be running or auto-startable.

Examples:
  probex test "if a user enters the wrong password 3 times the account should be locked"
  probex test "a non-admin user should not be able to access /admin endpoints"
  probex test "create an order via POST /orders then verify it with GET"
  probex test --dry-run "rate limit should return 429 when exceeding 100 req/s"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNLTest,
}

func init() {
	testNLCmd.Flags().Bool("dry-run", false, "generate tests but don't execute")
	testNLCmd.Flags().Int("concurrency", 5, "concurrent test executions")
	testNLCmd.Flags().Duration("timeout", 30*time.Second, "per-test timeout")

	rootCmd.AddCommand(testNLCmd)
}

func runNLTest(cmd *cobra.Command, args []string) error {
	description := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	fmt.Println(ui.Banner())
	bold.Println("\n  Natural Language Test Generation")
	fmt.Println()
	cyan.Printf("  Description: %s\n\n", description)

	// Load profile for endpoint context.
	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	profile, err := store.LoadProfile()
	if err != nil {
		ui.Warning("No API profile found. Tests will be generated without endpoint context.")
		profile = &models.APIProfile{}
	}

	// Start AI brain and generate.
	bold.Println("  Starting AI brain...")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	bridge := ai.NewBridge(0)
	if err := bridge.Start(ctx); err != nil {
		return fmt.Errorf("starting AI brain: %w", err)
	}
	defer bridge.Stop()

	client := ai.NewClient(bridge.Address())

	endpoints := ai.EndpointsToInfo(profile.Endpoints)

	nlReq := &ai.NLTestRequest{
		Description: description,
		Endpoints:   endpoints,
	}

	bold.Println("  Generating tests from description...")
	resp, err := client.NLToTest(ctx, nlReq)
	if err != nil {
		return fmt.Errorf("NL-to-test generation failed: %w", err)
	}

	tests := ai.GeneratedTestsToModelTests(resp.TestCases)
	if len(tests) == 0 {
		ui.Warning("AI couldn't generate tests from this description.")
		return nil
	}

	ui.Success(fmt.Sprintf("%d test cases generated from description", len(tests)))
	fmt.Println()

	// Display generated tests.
	for i, tc := range tests {
		fmt.Printf("  %d. [%s] %s\n", i+1, tc.Severity, tc.Name)
		if tc.Description != "" {
			fmt.Printf("     %s\n", tc.Description)
		}
	}
	fmt.Println()

	if dryRun {
		ui.Info("Dry run — skipping execution.")
		return nil
	}

	// Execute generated tests.
	bold.Println("  Executing generated tests...")
	fmt.Println()

	opts := models.RunOptions{
		Concurrency: concurrency,
		Timeout:     timeout,
	}

	exec := runner.New(opts)
	summary, err := exec.Execute(ctx, tests)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	fmt.Println()
	ui.RunSummary(summary)

	if err := store.SaveResults(summary); err != nil {
		ui.Warning(fmt.Sprintf("Failed to save results: %v", err))
	}

	if summary.Failed > 0 || summary.Errors > 0 {
		return fmt.Errorf("%d failed, %d errors", summary.Failed, summary.Errors)
	}
	return nil
}
