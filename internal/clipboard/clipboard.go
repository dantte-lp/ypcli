// Package clipboard is a thin, optional wrapper over the system clipboard. It
// shells out to platform tools (pbcopy/xclip/xsel/clip) via atotto/clipboard,
// requiring no CGO and degrading gracefully in headless environments such as
// CI, where Copy returns an error the caller can treat as non-fatal.
package clipboard

import (
	"errors"
	"fmt"

	"github.com/atotto/clipboard"
)

// errUnavailable is returned when no system clipboard utility is present.
var errUnavailable = errors.New("clipboard not available on this system")

// Indirections over atotto/clipboard so the behavior is unit-testable without a
// real clipboard.
var (
	unsupported = clipboard.Unsupported
	writeAll    = clipboard.WriteAll
)

// Available reports whether a system clipboard utility is usable.
func Available() bool {
	return !unsupported
}

// Copy writes s to the system clipboard.
func Copy(s string) error {
	if unsupported {
		return errUnavailable
	}
	if err := writeAll(s); err != nil {
		return fmt.Errorf("copy to clipboard: %w", err)
	}
	return nil
}
