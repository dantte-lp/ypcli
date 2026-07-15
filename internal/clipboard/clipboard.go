// Package clipboard is a thin, optional wrapper over the system clipboard. It
// shells out to platform tools (pbcopy/xclip/xsel/clip) via atotto/clipboard,
// requiring no CGO and degrading gracefully in headless environments such as
// CI, where Copy returns an error the caller can treat as non-fatal.
package clipboard

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// Available reports whether a system clipboard utility is usable.
func Available() bool {
	return !clipboard.Unsupported
}

// Copy writes s to the system clipboard.
func Copy(s string) error {
	if clipboard.Unsupported {
		return fmt.Errorf("clipboard not available on this system")
	}
	if err := clipboard.WriteAll(s); err != nil {
		return fmt.Errorf("copy to clipboard: %w", err)
	}
	return nil
}
