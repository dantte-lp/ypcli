package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/spf13/cobra"
)

func (a *app) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show client and server version",
		Args:  cobra.NoArgs,
		RunE:  a.runVersion,
	}
}

func (a *app) runVersion(cmd *cobra.Command, _ []string) error {
	s, err := a.resolve(cmd)
	if err != nil {
		return err
	}

	server := a.serverVersion(cmd, s)

	if s.jsonMode {
		return encodeJSON(cmd, map[string]string{
			"version": a.build.Version,
			"commit":  a.build.Commit,
			"date":    a.build.Date,
			"server":  server,
		})
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "ypcli %s (commit %s, built %s)\n", a.build.Version, a.build.Commit, a.build.Date)
	fmt.Fprintf(out, "server %s: %s\n", s.api, server)
	return nil
}

// serverVersion best-effort queries the server /version endpoint, returning a
// human label rather than failing the command on network/compat errors.
func (a *app) serverVersion(cmd *cobra.Command, s *settings) string {
	ctx, cancel := context.WithTimeout(cmd.Context(), s.timeout)
	defer cancel()

	token, err := config.ResolveToken(ctx, s.token, s.profile.TokenCommand)
	if err != nil {
		return "unknown"
	}
	v, err := newClient(s.api, token).Version(ctx)
	switch {
	case err == nil:
		return v
	case errors.Is(err, api.ErrUnsupported):
		return "unsupported (pre-13.x)"
	default:
		return "unreachable"
	}
}
