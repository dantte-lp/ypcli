// Command ypcli is a cross-platform CLI for publishing text and files as
// end-to-end-encrypted, self-expiring one-time secrets on a yopass server.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/dantte-lp/ypcli/internal/cli"
)

// Build metadata, injected at release time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Execute(ctx, cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
	stop()
	os.Exit(code)
}
