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
that can be consumed by the JS/TS and Java SDKs.

Examples:
  probex serve
  probex serve --addr localhost:9712`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")

		bold := color.New(color.Bold)
		fmt.Println(bold.Sprint("Starting PROBEX API server..."))
		fmt.Printf("  Listening on %s\n\n", color.CyanString("http://"+addr))

		srv := server.New(addr)

		// Graceful shutdown on SIGINT/SIGTERM.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start()
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
	rootCmd.AddCommand(serveCmd)
}
