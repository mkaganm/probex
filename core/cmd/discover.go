package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/scanner/iac"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/mkaganm/probex/internal/ui"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [directory]",
	Short: "Discover API endpoints from IaC files",
	Long: `Scan Infrastructure-as-Code files (Terraform, Pulumi, Kubernetes, Docker Compose)
to discover API endpoints without running a live scan.

Discovered endpoints are merged into the current profile.

Examples:
  probex discover .
  probex discover ./infrastructure
  probex discover ~/projects/my-api/terraform`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().Bool("merge", true, "merge discovered endpoints into existing profile")
	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	merge, _ := cmd.Flags().GetBool("merge")

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	fmt.Println(ui.Banner())
	bold.Println("\n  IaC Endpoint Discovery")
	fmt.Println()

	scanner := iac.New(dir)
	discovery, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("IaC scan failed: %w", err)
	}

	if len(discovery.Endpoints) == 0 {
		ui.Warning("No API endpoints found in IaC files.")
		return nil
	}

	ui.Success(fmt.Sprintf("Discovered %d endpoints from %s", len(discovery.Endpoints), discovery.Source))

	for _, f := range discovery.Files {
		cyan.Printf("  File: %s\n", f)
	}
	fmt.Println()

	for _, ep := range discovery.Endpoints {
		fmt.Printf("  %s %s", color.CyanString(ep.Method), ep.Path)
		if ep.BaseURL != "" {
			fmt.Printf(" (%s)", ep.BaseURL)
		}
		fmt.Println()
	}
	fmt.Println()

	if merge {
		store, err := storage.New("")
		if err != nil {
			return fmt.Errorf("storage init: %w", err)
		}

		profile, _ := store.LoadProfile()
		if profile == nil {
			ui.Warning("No existing profile. Create one first with 'probex scan' or manually.")
			return nil
		}

		added := 0
		existing := make(map[string]bool)
		for _, ep := range profile.Endpoints {
			existing[ep.Method+ep.Path] = true
		}
		for _, ep := range discovery.Endpoints {
			key := ep.Method + ep.Path
			if !existing[key] {
				profile.Endpoints = append(profile.Endpoints, ep)
				existing[key] = true
				added++
			}
		}

		if added > 0 {
			if err := store.SaveProfile(profile); err != nil {
				return fmt.Errorf("saving profile: %w", err)
			}
			ui.Success(fmt.Sprintf("Added %d new endpoints to profile (total: %d)", added, len(profile.Endpoints)))
		} else {
			ui.Info("All discovered endpoints already in profile.")
		}
	}

	return nil
}
