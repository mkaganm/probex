package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/probex/probex/internal/learn"
	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/storage"
	"github.com/probex/probex/internal/ui"
	"github.com/spf13/cobra"
)

var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Learn API behavior from traffic",
	Long: `Learn API behavior patterns from real traffic data.

Probex can learn from:
  - HAR files (browser/proxy captured traffic)
  - Live proxy traffic
  - Existing test logs

This builds a behavioral baseline for smarter test generation.

Examples:
  probex learn --from-traffic production.har
  probex learn --proxy localhost:8080
  probex learn --from-traffic ./logs/*.har`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromTraffic, _ := cmd.Flags().GetString("from-traffic")
		profileDir, _ := cmd.Flags().GetString("profile")

		if fromTraffic == "" {
			return fmt.Errorf("--from-traffic flag is required (path to HAR file or directory)")
		}

		fmt.Println(color.New(color.Bold).Sprint("Learning from traffic..."))
		fmt.Println()

		store, err := storage.New(profileDir)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}

		var profile *models.APIProfile
		var loadErr error
		if store.ProfileExists() {
			profile, loadErr = store.LoadProfile()
		}

		start := time.Now()
		learner := learn.NewLearner()
		result, err := learner.Learn(context.Background(), fromTraffic, profile)
		if err != nil {
			ui.Error(fmt.Sprintf("Learning failed: %v", err))
			return err
		}
		elapsed := time.Since(start)

		if err := store.SaveProfile(result.Profile); err != nil {
			ui.Error(fmt.Sprintf("Failed to save profile: %v", err))
			return err
		}

		fmt.Println()
		ui.Success(fmt.Sprintf("Analyzed %d HAR files, %d HTTP entries in %s",
			result.HARFilesRead, result.EntriesAnalyzed, elapsed.Round(time.Millisecond)))

		if loadErr == nil && profile != nil {
			ui.Info("Enriched existing profile")
		} else {
			ui.Info("Created new profile")
		}
		fmt.Println()

		if len(result.Profile.Endpoints) > 0 {
			fmt.Println(color.New(color.Bold).Sprint("  Discovered Endpoints"))
			fmt.Println()
			ui.EndpointList(result.Profile.Endpoints)
		}

		if result.TrafficAnalysis != nil {
			if len(result.TrafficAnalysis.Relationships) > 0 {
				fmt.Println(color.New(color.Bold).Sprint("  Endpoint Relationships"))
				fmt.Println()
				t := ui.NewTable("FROM", "TO", "TYPE", "COUNT")
				for _, rel := range result.TrafficAnalysis.Relationships {
					t.AddRow(rel.From, rel.To, string(rel.Type), fmt.Sprintf("%d", rel.Count))
				}
				fmt.Println(t.Render())
			}
		}

		if result.PatternReport != nil {
			totalPatterns := 0
			for _, patterns := range result.PatternReport.Endpoints {
				totalPatterns += len(patterns)
			}
			if totalPatterns > 0 {
				ui.Info(fmt.Sprintf("Detected %d field-level patterns across %d endpoints",
					totalPatterns, len(result.PatternReport.Endpoints)))
			}
		}

		if result.Profile.Baseline != nil {
			ui.Info(fmt.Sprintf("Built performance baselines for %d endpoints",
				len(result.Profile.Baseline.Endpoints)))
		}

		fmt.Println()
		ui.Success("Profile saved")

		return nil
	},
}

func init() {
	learnCmd.Flags().String("from-traffic", "", "path to HAR file or directory")
	learnCmd.Flags().String("proxy", "", "start proxy server on address (e.g. localhost:8080)")
	learnCmd.Flags().String("profile", "", "storage directory for API profile (default: .probex/)")

	rootCmd.AddCommand(learnCmd)
}
