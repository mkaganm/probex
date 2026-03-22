package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/probex/probex/internal/collective"
	"github.com/probex/probex/internal/storage"
	"github.com/probex/probex/internal/ui"
	"github.com/spf13/cobra"
)

var collectiveCmd = &cobra.Command{
	Use:   "collective",
	Short: "Community pattern sharing",
	Long: `Share anonymized test patterns with the PROBEX community and pull
patterns discovered by other users to enrich your local testing.

Privacy: Only abstract patterns are shared — never URLs, tokens, request bodies,
or any identifying information.`,
}

var collectivePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Share anonymized patterns with the community",
	RunE:  runCollectivePush,
}

var collectivePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull community patterns to enrich local tests",
	RunE:  runCollectivePull,
}

func init() {
	collectiveCmd.PersistentFlags().String("hub", "https://hub.probex.dev", "collective hub URL")
	collectivePullCmd.Flags().Float64("min-score", 0.7, "minimum pattern effectiveness score")
	collectivePullCmd.Flags().StringSlice("category", nil, "filter by categories")

	collectiveCmd.AddCommand(collectivePushCmd)
	collectiveCmd.AddCommand(collectivePullCmd)
	rootCmd.AddCommand(collectiveCmd)
}

func runCollectivePush(cmd *cobra.Command, args []string) error {
	hubURL, _ := cmd.Flags().GetString("hub")
	bold := color.New(color.Bold)

	fmt.Println(ui.Banner())
	bold.Println("\n  Collective Intelligence — Push")
	fmt.Println()

	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("storage init: %w", err)
	}

	results, err := store.LoadResults()
	if err != nil {
		if os.IsNotExist(err) {
			ui.Warning("No test results found. Run tests first.")
			return nil
		}
		return fmt.Errorf("loading results: %w", err)
	}

	anonymizer := collective.NewAnonymizer()
	patterns := anonymizer.ExtractPatterns(results)

	if len(patterns) == 0 {
		ui.Info("No patterns to share.")
		return nil
	}

	ui.Info(fmt.Sprintf("Anonymized %d patterns from latest run", len(patterns)))

	hostname, _ := os.Hostname()
	instanceID := collective.GenerateInstanceID(hostname)
	client := collective.NewClient(hubURL, instanceID)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	if err := client.Push(ctx, patterns); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	ui.Success(fmt.Sprintf("Shared %d patterns with the community", len(patterns)))
	return nil
}

func runCollectivePull(cmd *cobra.Command, args []string) error {
	hubURL, _ := cmd.Flags().GetString("hub")
	minScore, _ := cmd.Flags().GetFloat64("min-score")
	categories, _ := cmd.Flags().GetStringSlice("category")

	bold := color.New(color.Bold)

	fmt.Println(ui.Banner())
	bold.Println("\n  Collective Intelligence — Pull")
	fmt.Println()

	hostname, _ := os.Hostname()
	instanceID := collective.GenerateInstanceID(hostname)
	client := collective.NewClient(hubURL, instanceID)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	resp, err := client.Pull(ctx, categories, minScore)
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	if len(resp.Patterns) == 0 {
		ui.Info("No community patterns available matching your criteria.")
		return nil
	}

	ui.Success(fmt.Sprintf("Pulled %d community patterns (total available: %d)", len(resp.Patterns), resp.Total))
	fmt.Println()

	for _, p := range resp.Patterns {
		fmt.Printf("  [%.1f] %s — %s (%s)\n",
			p.Score,
			color.CyanString(p.TestType),
			p.Description,
			p.Severity,
		)
	}
	fmt.Println()

	ui.Info("Use 'probex run --collective' to include community patterns in your test suite.")
	return nil
}
