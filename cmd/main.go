package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/lavigneer/evergreen-lsp/pkg/lsp"
)

func main() {
	debugFlag := flag.CommandLine.Bool("v", false, "Sets logging to verbose")
	flag.Parse()
	logLevel := slog.LevelInfo
	if debugFlag != nil && *debugFlag {
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
}
