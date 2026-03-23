package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/mkaganm/probex/internal/ui"
	"github.com/mkaganm/probex/internal/watch"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously monitor API behavior",
	Long: `Watch mode runs continuously, monitoring API endpoints for:
  - Anomalous response times or status codes
  - Schema drift (response structure changes)
  - New or removed endpoints
  - Performance degradation

Examples:
  probex watch
  probex watch --interval 1m
  probex watch --notify slack:https://hooks.slack.com/xxx
  probex watch --notify webhook:https://example.com/hook`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().Duration("interval", 5*time.Minute, "polling interval")
	watchCmd.Flags().String("env", "", "environment label")
	watchCmd.Flags().String("notify", "", "notification target (stdout, slack:url, webhook:url)")
	watchCmd.Flags().StringSlice("endpoints", nil, "specific endpoints to watch")

	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	interval, _ := cmd.Flags().GetDuration("interval")
	env, _ := cmd.Flags().GetString("env")
	notify, _ := cmd.Flags().GetString("notify")
	endpoints, _ := cmd.Flags().GetStringSlice("endpoints")

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

	// Configure watch options
	opts := models.WatchOptions{
		Interval:  interval,
		Endpoints: endpoints,
	}

	// Parse alert targets
	targets := watch.ParseTargets(notify)
	alerter := watch.NewAlerter(targets...)

	// Banner
	fmt.Println(color.New(color.Bold).Sprint("PROBEX Watch Mode"))
	fmt.Println()
	ui.Info(fmt.Sprintf("Monitoring %d endpoints every %s", len(profile.Endpoints), interval))
	if env != "" {
		ui.Info(fmt.Sprintf("Environment: %s", env))
	}
	if len(endpoints) > 0 {
		ui.Info(fmt.Sprintf("Filtering: %v", endpoints))
	}
	fmt.Println()

	// Create watcher
	w := watch.New(profile, opts, alerter)

	// Set up event handler for terminal output
	cycleCount := 0
	w.OnEvent(func(event watch.WatchEvent) {
		cycleCount++
		ts := event.Timestamp.Format("15:04:05")
		if len(event.Anomalies) == 0 && len(event.Drifts) == 0 {
			fmt.Printf("[%s] Cycle %d: %d endpoints checked — all clear\n", ts, cycleCount, event.EndpointsChecked)
		} else {
			ui.Warning(fmt.Sprintf("[%s] Cycle %d: %d anomalies, %d drifts detected",
				ts, cycleCount, len(event.Anomalies), len(event.Drifts)))
		}
	})

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println()
		ui.Info("Shutting down watch mode...")
		cancel()
	}()

	ui.Info("Press Ctrl+C to stop")
	fmt.Println()

	err = w.Start(ctx)
	if err != nil && err != context.Canceled {
		return err
	}

	ui.Success("Watch mode stopped")
	return nil
}
