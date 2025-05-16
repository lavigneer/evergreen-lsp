package cmd

import (
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/mcp"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Runs an MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		done := make(chan struct{})
		// cwd, _ := os.Getwd()
		// workspaceRoot, err := config.FindWorkspaceRoot(cwd)
		// if err != nil {
		// 	return err
		// }
		workspaceRoot := "/Users/elavigne/workspace/mongo"

		// Have to do this to stop evergreen from logging...
		_ = grip.SetSender(send.NewMockSender("suppress"))

		cfg, err := config.NewWithDefaults(cmd.Context(), workspaceRoot)
		if err != nil {
			return err
		}
		server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

		for _, p := range cfg.Projects {
			mcpExecutor := mcp.New(p)
			mcpExecutor.Register(server)
		}

		err = server.Serve()
		if err != nil {
			return err
		}
		<-done
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
