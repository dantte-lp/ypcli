package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/spf13/cobra"
)

// newClient builds an API client, attaching the token only when present.
func newClient(baseAPI, token string) *api.Client {
	if token == "" {
		return api.New(baseAPI)
	}
	return api.New(baseAPI, api.WithToken(token))
}

func stringReader(s string) io.Reader { return strings.NewReader(s) }

// encodeJSON writes v as indented JSON to the command's stdout.
func encodeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
