package cmd

import (
	"log/slog"
	"os"

	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/lint"
	"github.com/lavigneer/evergreen-lsp/pkg/reporter"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
	"github.com/spf13/cobra"
)

// lintCmd represents the lint command
var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint an evergreen project",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, _ := os.Getwd()
		workspaceRoot, err := config.FindWorkspaceRoot(cwd)
		if err != nil {
			return err
		}

		// Have to do this to stop evergreen from logging...
		_ = grip.SetSender(send.NewMockSender("suppress"))

		cfg, err := config.NewWithDefaults(cmd.Context(), workspaceRoot)
		if err != nil {
			return err
		}
		rep := reporter.Default{}
		for _, p := range cfg.Projects {
			lintExecutor := lint.New(p, cfg.Lint)
			diagnostics, err := lintExecutor.Lint()
			if err != nil {
				slog.Error("Could not lint project", "project", p.Path())
			}
			rep.ReportDiagnostics(cmd.Context(), diagnostics)

		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
}
