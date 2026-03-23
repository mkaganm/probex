package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/mkaganm/probex/internal/models"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage probex configuration",
	Long:  `View and manage probex configuration.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new probex configuration file",
	Long: `Create a new probex.yaml configuration file in the current directory
with sensible defaults.

Examples:
  probex config init
  probex config init --output probex.yaml
  probex config init --target https://api.example.com`,
	RunE: runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)

	configInitCmd.Flags().StringP("output", "o", "probex.yaml", "output config file path")
	configInitCmd.Flags().String("target", "", "target API base URL")

	rootCmd.AddCommand(configCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	outputPath, _ := cmd.Flags().GetString("output")
	targetURL, _ := cmd.Flags().GetString("target")

	// Check if file already exists.
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("config file %s already exists — use --output to specify a different path", outputPath)
	}

	cfg := models.DefaultConfig()

	if targetURL != "" {
		cfg.Target.BaseURL = targetURL
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Add header comment.
	header := "# PROBEX Configuration\n" +
		"# See https://github.com/mkaganm/probex for documentation\n" +
		"#\n" +
		"# Quick start:\n" +
		"#   1. Set target.base_url to your API\n" +
		"#   2. Run: probex scan\n" +
		"#   3. Run: probex run\n" +
		"#\n\n"

	content := header + string(data)

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	fmt.Println()
	bold.Println("  PROBEX Configuration Initialized")
	fmt.Println()
	green.Printf("  Created %s\n", outputPath)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    1. Edit %s and set your target API URL\n", cyan.Sprint(outputPath))
	fmt.Printf("    2. Run %s to discover endpoints\n", cyan.Sprint("probex scan"))
	fmt.Printf("    3. Run %s to execute tests\n", cyan.Sprint("probex run"))
	fmt.Println()

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold)

	// Try to load config from file.
	configFile := "probex.yaml"
	if cfgFile != "" {
		configFile = cfgFile
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		// No config file; show defaults.
		bold.Println("  No config file found — showing defaults:")
		fmt.Println()
		cfg := models.DefaultConfig()
		yamlData, _ := yaml.Marshal(cfg)
		fmt.Println(string(yamlData))
		return nil
	}

	bold.Printf("  Configuration from %s:\n", configFile)
	fmt.Println()
	fmt.Println(string(data))
	return nil
}
