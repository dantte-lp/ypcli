package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/dantte-lp/ypcli/internal/output"
	"github.com/dantte-lp/ypcli/internal/share"
	"github.com/spf13/cobra"
)

func (a *app) newReceiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "receive [url]",
		Short: "Fetch and decrypt a secret",
		Long: "Fetch a secret by its share URL (or --id/--key) and decrypt it. Text is\n" +
			"written to stdout; files are written to their original name or --output.",
		Example: "  ypcli receive 'https://yopass.se/#/s/ID/KEY'\n" +
			"  ypcli receive --id ID --key KEY --file -o ./out/\n" +
			"  ypcli receive 'https://.../#/c/ID' --key MANUALKEY",
		Args: cobra.MaximumNArgs(1),
		RunE: a.runReceive,
	}
	f := cmd.Flags()
	f.String("id", "", "secret ID (when no URL is given)")
	f.StringP("key", "k", "", "decryption key")
	f.Bool("file", false, "treat the secret as a file (with --id)")
	f.StringP("output", "o", "", "output file or directory (files)")
	return cmd
}

func (a *app) runReceive(cmd *cobra.Command, args []string) error {
	s, err := a.resolve(cmd)
	if err != nil {
		return err
	}

	id, key, fileOpt, err := receiveTarget(cmd, args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), s.timeout)
	defer cancel()

	token, err := config.ResolveToken(ctx, s.token, s.profile.TokenCommand)
	if err != nil {
		return err
	}
	client := newClient(s.api, token)
	s.log.Debug("receiving secret", "api", s.api, "id", id, "file", fileOpt, "authenticated", token != "")

	if fileOpt {
		return a.receiveFile(ctx, cmd, s, client, id, key)
	}
	return a.receiveText(ctx, cmd, s, client, id, key)
}

func (a *app) receiveText(ctx context.Context, cmd *cobra.Command, s *settings, client *api.Client, id, key string) error {
	res, err := share.Receive(ctx, client, share.Target{ID: id, Key: key})
	if err != nil {
		return err
	}

	if out, _ := cmd.Flags().GetString("output"); out != "" {
		if err := writeFile(out, []byte(res.Content)); err != nil {
			return err
		}
		return s.printer(cmd).Receive(output.ReceiveResult{Written: out, Bytes: len(res.Content)})
	}
	return s.printer(cmd).Receive(output.ReceiveResult{Content: res.Content})
}

func (a *app) receiveFile(ctx context.Context, cmd *cobra.Command, s *settings, client *api.Client, id, key string) error {
	target := share.Target{ID: id, Key: key, File: true}
	if !s.jsonMode {
		target.Wrap = func(r io.Reader, total int64) io.Reader {
			return output.NewProgressReader(r, total, cmd.ErrOrStderr(), "downloading")
		}
	}

	res, err := share.Receive(ctx, client, target)
	if err != nil {
		return err
	}

	out, _ := cmd.Flags().GetString("output")
	dest := destPath(out, res.Filename, id)
	if err := writeFile(dest, []byte(res.Content)); err != nil {
		return err
	}
	return s.printer(cmd).Receive(output.ReceiveResult{
		Written: dest, Filename: res.Filename, Bytes: len(res.Content),
	})
}

// receiveTarget derives (id, key, fileOpt) from a positional URL or the
// --id/--key/--file flags. It errors (usage) when required inputs are missing.
func receiveTarget(cmd *cobra.Command, args []string) (id, key string, fileOpt bool, err error) {
	keyFlag, _ := cmd.Flags().GetString("key")

	if len(args) == 1 {
		var keyOpt bool
		id, key, fileOpt, keyOpt, err = crypto.ParseURL(args[0])
		if err != nil {
			return "", "", false, usage("%v", err)
		}
		if keyOpt || key == "" {
			if keyFlag == "" {
				return "", "", false, usage("this link needs a manual key: pass --key")
			}
			key = keyFlag
		}
		return id, key, fileOpt, nil
	}

	id, _ = cmd.Flags().GetString("id")
	if id == "" {
		return "", "", false, usage("provide a share URL or --id")
	}
	if keyFlag == "" {
		return "", "", false, usage("--key is required with --id")
	}
	fileOpt, _ = cmd.Flags().GetBool("file")
	return id, keyFlag, fileOpt, nil
}

// destPath resolves where to write a received file. An empty out uses the
// embedded filename; a directory out (existing, or written with a trailing
// separator) joins the filename; otherwise out is the exact path. id is a
// last-resort filename.
func destPath(out, filename, id string) string {
	name := filename
	if name == "" {
		name = id
	}
	if out == "" {
		return name
	}
	if looksLikeDir(out) {
		return filepath.Join(out, name)
	}
	return out
}

// looksLikeDir reports whether out should be treated as a directory: it either
// exists as one, or ends with a path separator (directory intent).
func looksLikeDir(out string) bool {
	if strings.HasSuffix(out, "/") || strings.HasSuffix(out, string(os.PathSeparator)) {
		return true
	}
	info, err := os.Stat(out)
	return err == nil && info.IsDir()
}

// writeFile writes data to path (mode 0600), creating parent directories.
func writeFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
