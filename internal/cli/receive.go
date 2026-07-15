package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/dantte-lp/ypcli/internal/output"
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

	if fileOpt {
		return a.receiveFile(ctx, cmd, s, client, id, key)
	}
	return a.receiveText(ctx, cmd, s, client, id, key)
}

func (a *app) receiveText(ctx context.Context, cmd *cobra.Command, s *settings, client *api.Client, id, key string) error {
	msg, err := client.FetchSecret(ctx, id)
	if err != nil {
		return err
	}
	plaintext, _, err := crypto.Decrypt(stringReader(msg), key)
	if err != nil {
		return err
	}

	if out, _ := cmd.Flags().GetString("output"); out != "" {
		if err := os.WriteFile(out, []byte(plaintext), 0o600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return s.printer(cmd).Receive(output.ReceiveResult{Written: out, Bytes: len(plaintext)})
	}
	return s.printer(cmd).Receive(output.ReceiveResult{Content: plaintext})
}

func (a *app) receiveFile(ctx context.Context, cmd *cobra.Command, s *settings, client *api.Client, id, key string) error {
	body, size, err := client.FetchFile(ctx, id)
	if err != nil {
		return err
	}
	defer body.Close()

	var src io.Reader = body
	if !s.jsonMode {
		src = output.NewProgressReader(body, size, cmd.ErrOrStderr(), "downloading")
	}

	plaintext, filename, err := crypto.Decrypt(src, key)
	if err != nil {
		return err
	}

	out, _ := cmd.Flags().GetString("output")
	dest := destPath(out, filename, id)
	if err := os.WriteFile(dest, []byte(plaintext), 0o600); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return s.printer(cmd).Receive(output.ReceiveResult{
		Written: dest, Filename: filename, Bytes: len(plaintext),
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
// embedded filename; a directory out joins the filename; otherwise out is the
// exact path. id is a last-resort filename.
func destPath(out, filename, id string) string {
	name := filename
	if name == "" {
		name = id
	}
	if out == "" {
		return name
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, name)
	}
	return out
}
