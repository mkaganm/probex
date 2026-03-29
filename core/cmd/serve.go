package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/mkaganm/probex/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start REST API server for SDK integration",
	Long: `Start a local REST API server that SDK clients can connect to.

The server exposes endpoints for scanning, running tests, and retrieving results
that can be consumed by the JS/TS, Java, and Kotlin SDKs.

Use --ai to start the Python AI brain alongside the server, enabling AI-powered
endpoints under /api/v1/ai/*. Alternatively, use --ai-url to connect to an
externally managed brain service.

Examples:
  probex serve
  probex serve --addr localhost:9712
  probex serve --ai
  probex serve --ai-url http://localhost:9711`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		enableAI, _ := cmd.Flags().GetBool("ai")
		aiPort, _ := cmd.Flags().GetInt("ai-port")
		aiURL, _ := cmd.Flags().GetString("ai-url")

		bold := color.New(color.Bold)
		fmt.Println(bold.Sprint("Starting PROBEX API server..."))
		fmt.Printf("  Listening on %s\n", color.CyanString("http://"+addr))

		var opts []server.Option
		switch {
		case aiURL != "":
			fmt.Printf("  AI brain:  %s (external)\n", color.GreenString(aiURL))
			opts = append(opts, server.WithAIURL(aiURL))
		case enableAI:
			fmt.Printf("  AI brain:  %s (managed)\n", color.GreenString("enabled"))
			opts = append(opts, server.WithAI(aiPort))
		default:
			fmt.Printf("  AI brain:  %s\n", color.YellowString("disabled (use --ai to enable)"))
		}
		fmt.Println()

		srv, err := server.New(addr, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Graceful shutdown on SIGINT/SIGTERM.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start(ctx)
		}()

		select {
		case err := <-errCh:
			return fmt.Errorf("server error: %w", err)
		case <-ctx.Done():
			fmt.Println("\nShutting down server...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5s
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		}
	},
}

func init() {
	serveCmd.Flags().String("addr", "localhost:9712", "address to listen on")
	serveCmd.Flags().Bool("ai", false, "start AI brain alongside the server")
	serveCmd.Flags().Int("ai-port", 0, "AI brain port (default 9711)")
	serveCmd.Flags().String("ai-url", "", "connect to an external AI brain URL")
	rootCmd.AddCommand(serveCmd)
}
