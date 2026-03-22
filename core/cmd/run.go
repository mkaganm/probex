package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/probex/probex/internal/ai"
	"github.com/probex/probex/internal/generator"
	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/runner"
	"github.com/probex/probex/internal/storage"
	"github.com/probex/probex/internal/ui"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Generate and execute tests",
	Long: `Generate and run tests against a previously scanned API profile.

Tests are auto-generated based on discovered endpoints and learned behavior.
No test code needed.

Examples:
  probex run
  probex run --profile ./my-api.probex.json
  probex run --category security,edge_case
  probex run --ai --concurrency 20`,
	RunE: runTests,
}

func init() {
	runCmd.Flags().String("profile", "", "path to API profile (default: .probex/profile.json)")
	runCmd.Flags().StringSlice("category", nil, "test categories to run (happy_path,edge_case,security,fuzz)")
	runCmd.Flags().Int("concurrency", 5, "number of concurrent test executions")
	runCmd.Flags().Bool("ai", false, "enable AI-powered test generation")
	runCmd.Flags().Bool("stop-on-fail", false, "stop execution on first failure")
	runCmd.Flags().Duration("timeout", 30*time.Second, "per-test timeout")

	rootCmd.AddCommand(runCmd)
}

func runTests(cmd *cobra.Command, args []string) error {
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	stopOnFail, _ := cmd.Flags().GetBool("stop-on-fail")
	categories, _ := cmd.Flags().GetStringSlice("category")
	useAI, _ := cmd.Flags().GetBool("ai")

	bold := color.New(color.Bold)

	fmt.Println(ui.Banner())
	bold.Println("\n  Running tests...")
	fmt.Println()

	// Load profile
	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	profile, err := store.LoadProfile()
	if err != nil {
		ui.Error("No API profile found. Run 'probex scan <url>' first.")
		return fmt.Errorf("failed to load profile: %w", err)
	}

	ui.Info(fmt.Sprintf("Loaded profile: %s (%d endpoints)", profile.BaseURL, len(profile.Endpoints)))
	fmt.Println()

	// Generate tests
	bold.Println("  Generating tests...")
	eng := generator.New(profile)

	// Filter categories if specified
	if len(categories) > 0 {
		catMap := make(map[models.TestCategory]bool)
		for _, c := range categories {
			catMap[models.TestCategory(c)] = true
		}
		eng.SetCategoryFilter(catMap)
	}

	tests, err := eng.Generate()
	if err != nil {
		return fmt.Errorf("test generation failed: %w", err)
	}

	ui.Success(fmt.Sprintf("%d test cases generated", len(tests)))
	fmt.Println()

	// AI-powered test generation
	if useAI {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		aiTests, aiErr := generateAITests(ctx, profile)
		cancel()
		if aiErr != nil {
			ui.Warning(fmt.Sprintf("AI test generation failed: %v", aiErr))
		} else {
			tests = append(tests, aiTests...)
			ui.Success(fmt.Sprintf("AI generated %d additional test cases", len(aiTests)))
		}
		fmt.Println()
	}

	if len(tests) == 0 {
		ui.Warning("No tests generated. The profile may have no endpoints.")
		return nil
	}

	// Execute tests
	bold.Println("  Executing tests...")
	fmt.Println()

	opts := models.RunOptions{
		Concurrency: concurrency,
		Timeout:     timeout,
		StopOnFail:  stopOnFail,
	}

	exec := runner.New(opts)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	summary, err := exec.Execute(ctx, tests)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Display results
	fmt.Println()
	ui.RunSummary(summary)

	// Save results
	if err := store.SaveResults(summary); err != nil {
		ui.Warning(fmt.Sprintf("Failed to save results: %v", err))
	} else {
		ui.Info("Results saved to .probex/")
	}

	// Exit with non-zero if failures
	if summary.Failed > 0 || summary.Errors > 0 {
		return fmt.Errorf("%d failed, %d errors", summary.Failed, summary.Errors)
	}

	return nil
}

// generateAITests starts the Python brain, generates AI test scenarios, and
// converts them to models.TestCase. It handles bridge lifecycle internally.
func generateAITests(ctx context.Context, profile *models.APIProfile) ([]models.TestCase, error) {
	bridge := ai.NewBridge(0)

	if err := bridge.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting AI brain: %w", err)
	}
	defer bridge.Stop()

	client := ai.NewClient(bridge.Address())

	// Convert profile endpoints to AI endpoint info.
	endpoints := ai.EndpointsToInfo(profile.Endpoints)

	req := &ai.ScenarioRequest{
		Endpoints:      endpoints,
		ProfileContext: fmt.Sprintf("API: %s (%d endpoints)", profile.BaseURL, len(profile.Endpoints)),
		MaxScenarios:   len(profile.Endpoints) * 5, // up to 5 scenarios per endpoint
	}

	resp, err := client.GenerateScenarios(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generating scenarios: %w", err)
	}

	tests := ai.GeneratedTestsToModelTests(resp.Scenarios)
	return tests, nil
}
