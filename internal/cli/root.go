// Package cli wires the ypcli command tree, global flags, configuration
// resolution and the central error-to-exit-code mapping.
package cli

import "context"

// BuildInfo carries release metadata injected into the binary at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Execute runs the ypcli command tree and returns the process exit code.
//
// The full command tree is wired in Phase 5; this stub keeps the module
// buildable while the lower layers are implemented.
func Execute(ctx context.Context, _ BuildInfo) int {
	_ = ctx
	return 0
}
