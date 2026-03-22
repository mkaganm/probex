package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/probex/probex/internal/proxy"
	"github.com/probex/probex/internal/storage"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy <target-url>",
	Short: "Start a reverse proxy to capture API traffic",
	Long: `Start a transparent reverse proxy that forwards requests to the target API
while capturing all traffic for learning.

The captured traffic can be exported as HAR or used to build an API profile.

Examples:
  probex proxy http://localhost:8080
  probex proxy http://localhost:8080 --listen :9090
  probex proxy http://localhost:8080 --export har.json`,
	Args: cobra.ExactArgs(1),
	RunE: runProxy,
}

func init() {
	proxyCmd.Flags().StringP("listen", "l", ":9090", "proxy listen address")
	proxyCmd.Flags().String("export", "", "export captured traffic to HAR file on shutdown")
	proxyCmd.Flags().Bool("learn", false, "automatically learn and build profile from captured traffic")
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) error {
	targetURL := args[0]
	listenAddr, _ := cmd.Flags().GetString("listen")
	exportPath, _ := cmd.Flags().GetString("export")
	autoLearn, _ := cmd.Flags().GetBool("learn")

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)

	fmt.Println()
	bold.Println("  PROBEX Proxy")
	fmt.Println()
	fmt.Printf("  Target:  %s\n", cyan.Sprint(targetURL))
	fmt.Printf("  Listen:  %s\n", cyan.Sprint(listenAddr))
	if exportPath != "" {
		fmt.Printf("  Export:  %s\n", cyan.Sprint(exportPath))
	}
	if autoLearn {
		fmt.Printf("  Mode:    %s\n", green.Sprint("auto-learn"))
	}
	fmt.Println()
	fmt.Println("  Proxying traffic... Press Ctrl+C to stop.")
	fmt.Println()

	p, err := proxy.New(proxy.Config{
		ListenAddr: listenAddr,
		TargetURL:  targetURL,
		OnEvent: func(c proxy.CapturedRequest) {
			statusColor := color.New(color.FgGreen)
			if c.StatusCode >= 400 {
				statusColor = color.New(color.FgRed)
			} else if c.StatusCode >= 300 {
				statusColor = color.New(color.FgYellow)
			}
			fmt.Printf("  %s %s → %s (%s)\n",
				color.New(color.FgCyan).Sprintf("%-6s", c.Method),
				c.Path,
				statusColor.Sprintf("%d", c.StatusCode),
				c.Duration.Round(1000000), // ms precision
			)
		},
	})
	if err != nil {
		return fmt.Errorf("create proxy: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println()
		bold.Println("  Shutting down proxy...")

		count := p.CaptureCount()
		fmt.Printf("  Captured %d requests\n", count)

		// Export HAR if requested.
		if exportPath != "" && count > 0 {
			harData, err := p.ExportHAR()
			if err != nil {
				color.Red("  Failed to export HAR: %v", err)
			} else {
				if err := os.WriteFile(exportPath, harData, 0o644); err != nil {
					color.Red("  Failed to write HAR: %v", err)
				} else {
					green.Printf("  Exported HAR to %s\n", exportPath)
				}
			}
		}

		// Auto-learn if requested.
		if autoLearn && count > 0 {
			profile := p.ToAPIProfile()
			if profile != nil {
				store, err := storage.New("")
				if err == nil {
					if err := store.SaveProfile(profile); err != nil {
						color.Red("  Failed to save profile: %v", err)
					} else {
						green.Printf("  Saved profile with %d endpoints\n", len(profile.Endpoints))
					}
				}
			}
		}

		cancel()
	}()

	if err := p.Start(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("proxy server: %w", err)
	}

	return nil
}
