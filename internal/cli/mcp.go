package cli

import (
	"fmt"

	"github.com/dantte-lp/ypcli/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func (a *app) newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server exposing ypcli to AI agents",
		Long: "Serve ypcli's send/receive operations over the Model Context Protocol\n" +
			"so agents (Claude, Codex, Gemini, …) can share and fetch secrets. Uses\n" +
			"stdio by default, or HTTP with --http for a shared server.",
		Args: cobra.NoArgs,
		RunE: a.runMCP,
	}
	f := cmd.Flags()
	f.String("http", "", "serve over HTTP at this address instead of stdio (e.g. 127.0.0.1:8765)")
	f.String("http-token", "", "bearer token required in HTTP mode ($YPCLI_MCP_TOKEN)")
	f.Bool("read-only", false, "expose send-only tools (omit receive_secret)")
	return cmd
}

func (a *app) runMCP(cmd *cobra.Command, _ []string) error {
	readOnly, _ := cmd.Flags().GetBool("read-only")
	cfgPath, err := configPath(cmd.Root())
	if err != nil {
		return err
	}
	srv := mcpserver.New(mcpserver.Options{
		ConfigPath: cfgPath,
		ReadOnly:   readOnly,
		Version:    a.build.Version,
	})

	if addr, _ := cmd.Flags().GetString("http"); addr != "" {
		return fmt.Errorf("--http mode is not implemented yet")
	}
	return srv.Run(cmd.Context(), &mcp.StdioTransport{})
}
