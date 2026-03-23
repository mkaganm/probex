package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mkaganm/probex/internal/graph"
	"github.com/mkaganm/probex/internal/storage"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize endpoint relationships",
	Long: `Display a graph of endpoint relationships discovered from the API profile.

Supported output formats:
  - ascii (default): Terminal-friendly ASCII art
  - dot: Graphviz DOT format for rendering with graphviz

Examples:
  probex graph
  probex graph --format dot > api.dot
  probex graph --format dot | dot -Tpng -o api.png`,
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().String("format", "ascii", "output format: ascii, dot")
	graphCmd.Flags().StringP("output", "o", "", "write output to file instead of stdout")
	rootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")

	store, err := storage.New("")
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}

	if !store.ProfileExists() {
		return fmt.Errorf("no API profile found — run 'probex scan' first")
	}

	profile, err := store.LoadProfile()
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}

	g := graph.New(profile)
	g.InferEdges()

	var output string
	switch format {
	case "ascii":
		output = g.RenderASCII()
	case "dot":
		output = g.RenderDOT()
	default:
		return fmt.Errorf("unknown format %q (use: ascii, dot)", format)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0o644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		color.Green("Graph written to %s", outputFile)
		return nil
	}

	fmt.Print(output)
	return nil
}
