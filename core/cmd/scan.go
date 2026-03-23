package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/scanner"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/mkaganm/probex/internal/ui"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan <url>",
	Short: "Discover API endpoints",
	Long: `Scan and discover all API endpoints from a base URL.

Probex will:
  - Look for OpenAPI/Swagger specs
  - Crawl and discover endpoints
  - Probe common API paths
  - Detect authentication requirements
  - Infer request/response schemas

Examples:
  probex scan https://api.example.com
  probex scan https://api.example.com --auth-header "Bearer token123"
  probex scan https://api.example.com --depth 5 --concurrency 20`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	scanCmd.Flags().String("auth-header", "", "authorization header value (e.g. \"Bearer token\")")
	scanCmd.Flags().Int("depth", 3, "maximum crawl depth")
	scanCmd.Flags().Int("concurrency", 10, "number of concurrent requests")
	scanCmd.Flags().Duration("timeout", 30*time.Second, "request timeout")
	scanCmd.Flags().String("wordlist", "", "custom path wordlist file")
	scanCmd.Flags().StringP("output", "o", "", "save profile to file")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	targetURL := args[0]
	depth, _ := cmd.Flags().GetInt("depth")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	authHeader, _ := cmd.Flags().GetString("auth-header")

	bold := color.New(color.Bold)

	fmt.Println(ui.Banner())
	bold.Printf("\n  Scanning %s\n\n", targetURL)

	opts := models.ScanOptions{
		MaxDepth:    depth,
		Concurrency: concurrency,
		Timeout:     timeout,
		FollowLinks: true,
	}

	s := scanner.New(targetURL, opts)
	if authHeader != "" {
		s.SetAuth(authHeader)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	start := time.Now()
	result, err := s.Scan(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Scan failed: %v", err))
		return err
	}
	duration := time.Since(start)

	if len(result.Endpoints) == 0 {
		ui.Warning("No endpoints discovered. Try with --auth-header or a different URL.")
		return nil
	}

	ui.EndpointList(result.Endpoints)
	ui.ScanSummary(len(result.Endpoints), duration)

	// Save profile
	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	profile := &models.APIProfile{
		ID:        fmt.Sprintf("scan_%d", time.Now().Unix()),
		Name:      targetURL,
		BaseURL:   targetURL,
		Endpoints: result.Endpoints,
		Auth:      result.Auth,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ScanConfig: models.ScanConfig{
			MaxDepth:    depth,
			Timeout:     timeout,
			Concurrency: concurrency,
			AuthHeader:  authHeader,
		},
	}

	if err := store.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	ui.Success("Profile saved to .probex/profile.json")
	return nil
}
