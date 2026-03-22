package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "probex",
	Short: "Zero-Test API Intelligence Engine",
	Long: color.New(color.FgCyan, color.Bold).Sprint("PROBEX") + ` — Zero-Test API Intelligence Engine

Probex discovers, learns, and autonomously tests your APIs.
No test code needed — just point it at your API and let it work.

  $ probex scan https://api.example.com
  $ probex run
  $ probex watch --env staging`,
	SilenceUsage: true,
	Version:      "1.0.0",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./probex.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
