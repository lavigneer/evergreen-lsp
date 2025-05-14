package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/lavigneer/evergreen-lsp/pkg/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Execute the evergreen lsp",
	Run: func(cmd *cobra.Command, _ []string) {
		debugFlag, _ := cmd.Flags().GetBool("verbose")
		logLevel := slog.LevelInfo
		if debugFlag {
			logLevel = slog.LevelDebug
		}

		// Set slog to log to stderr instead of stdout since we are using stdio for the server
		logHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
		slog.SetDefault(slog.New(logHandler))
		logger := slog.NewLogLogger(logHandler, slog.LevelInfo)

		// Set up lsp handler and start
		slog.Info("Setting up evergreen lsp")
		handler := lsp.NewHandler()
		<-lsp.New(handler, logger).Start(context.Background())
		slog.Info("Connection closed")
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	lspCmd.Flags().BoolP("verbose", "v", false, "Sets logging to verbose")
}
