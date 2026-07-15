package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// openEditor launches the user's editor on a temporary file and returns its
// contents. The editor is resolved from YPCLI_EDITOR, VISUAL, or EDITOR, with a
// platform fallback. The editor command may include arguments (e.g. "code
// --wait"); the temp file path is appended as the final argument.
func openEditor(ctx context.Context) (string, error) {
	editor := coalesce(os.Getenv("YPCLI_EDITOR"), os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	tmp, err := os.CreateTemp("", "ypcli-secret-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	name := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(name) //nolint:errcheck // best-effort cleanup of a temp file

	fields := strings.Fields(editor)
	args := make([]string, 0, len(fields))
	args = append(args, fields[1:]...)
	args = append(args, name)
	cmd := exec.CommandContext(ctx, fields[0], args...) //nolint:gosec // editor is user-configured
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q exited: %w", editor, err)
	}

	data, err := os.ReadFile(name) //nolint:gosec // name is our own temp file
	if err != nil {
		return "", fmt.Errorf("read edited file: %w", err)
	}
	return string(data), nil
}
