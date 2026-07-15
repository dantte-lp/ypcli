package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/clipboard"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/dantte-lp/ypcli/internal/output"
	"github.com/spf13/cobra"
)

func (a *app) newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Encrypt and publish a secret (text or file)",
		Long: "Encrypt text or a file client-side and publish it to a yopass server,\n" +
			"printing a one-time share URL. Input is taken from --file, --text, or stdin.",
		Example: "  printf 'secret' | ypcli send --one-time\n" +
			"  ypcli send --file db.env --expiration 1d --json\n" +
			"  echo hi | ypcli send --profile work --require-auth",
		Args: cobra.NoArgs,
		RunE: a.runSend,
	}
	f := cmd.Flags()
	f.StringP("file", "f", "", "read secret from a file")
	f.StringP("text", "t", "", "secret text (instead of stdin/file)")
	f.StringP("expiration", "e", "", "lifetime: 1h, 1d or 1w")
	f.Bool("one-time", true, "delete after first view")
	f.Bool("require-auth", false, "require authentication to view (server support needed)")
	f.StringP("key", "k", "", "manual encryption key (omitted from the URL)")
	f.Bool("qr", false, "also render the URL as a terminal QR code")
	f.Bool("copy", false, "copy the URL to the system clipboard")
	return cmd
}

func (a *app) runSend(cmd *cobra.Command, _ []string) error {
	s, err := a.resolve(cmd)
	if err != nil {
		return err
	}

	exp, err := resolveExpiration(cmd, s.profile)
	if err != nil {
		return err
	}
	oneTime := resolveOneTime(cmd, s.profile)
	requireAuth, _ := cmd.Flags().GetBool("require-auth")

	key, manualKey, err := resolveKey(cmd)
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
	useArgon2 := resolveArgon2(ctx, client, s.profile)

	filePath, _ := cmd.Flags().GetString("file")
	var (
		id      string
		fileOpt bool
	)
	if filePath != "" {
		id, err = sendFile(ctx, client, filePath, key, exp, oneTime, useArgon2)
		fileOpt = true
	} else {
		id, err = sendText(ctx, cmd, client, key, exp, oneTime, requireAuth, useArgon2)
	}
	if err != nil {
		return err
	}

	shareURL := crypto.SecretURL(s.url, id, key, fileOpt, manualKey)
	return emitSend(cmd, s, output.SendResult{
		ID: id, URL: shareURL, Key: key, ManualKey: manualKey,
		File: fileOpt, OneTime: oneTime, Expiration: expirationLabel(exp),
	})
}

func sendText(ctx context.Context, cmd *cobra.Command, client *api.Client, key string, exp int32, oneTime, requireAuth, argon2 bool) (string, error) {
	r, err := textReader(cmd)
	if err != nil {
		return "", err
	}
	enc := crypto.Encrypt
	if argon2 {
		enc = crypto.EncryptWithArgon2
	}
	msg, err := enc(r, key)
	if err != nil {
		return "", fmt.Errorf("encrypt secret: %w", err)
	}
	return client.CreateSecret(ctx, api.Secret{
		Message: msg, Expiration: exp, OneTime: oneTime, RequireAuth: requireAuth,
	})
}

func sendFile(ctx context.Context, client *api.Client, path, key string, exp int32, oneTime, argon2 bool) (string, error) {
	f, err := os.Open(path) //nolint:gosec // path provided by the user by design
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	encBin := crypto.EncryptBinary
	if argon2 {
		encBin = crypto.EncryptBinaryWithArgon2
	}
	data, err := encBin(f, key, filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("encrypt file: %w", err)
	}
	return client.CreateFile(ctx, readerOf(data), exp, oneTime)
}

// textReader returns the plaintext source: --text, else stdin (must be piped).
func textReader(cmd *cobra.Command) (io.Reader, error) {
	if text, _ := cmd.Flags().GetString("text"); text != "" {
		return stringReader(text), nil
	}
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return nil, fmt.Errorf("stat stdin: %w", err)
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return nil, usage("no input: provide --file, --text or piped stdin")
		}
	}
	return in, nil
}

func emitSend(cmd *cobra.Command, s *settings, res output.SendResult) error {
	if copyFlag, _ := cmd.Flags().GetBool("copy"); copyFlag {
		if err := clipboard.Copy(res.URL); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", err)
		}
	}
	if qr, _ := cmd.Flags().GetBool("qr"); qr && !s.jsonMode {
		if art, err := output.QR(res.URL); err == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), art)
		}
	}
	return s.printer(cmd).Send(res)
}

// resolveExpiration picks the expiration flag if set, else the profile value,
// else the default, and validates it.
func resolveExpiration(cmd *cobra.Command, p config.Profile) (int32, error) {
	label := coalesce(changedString(cmd, "expiration"), p.Expiration, config.DefaultExpiration)
	seconds, ok := crypto.ExpirationSeconds(label)
	if !ok {
		return 0, usage("invalid expiration %q: use 1h, 1d or 1w", label)
	}
	return seconds, nil
}

func resolveOneTime(cmd *cobra.Command, p config.Profile) bool {
	if cmd.Flags().Changed("one-time") {
		v, _ := cmd.Flags().GetBool("one-time")
		return v
	}
	if p.OneTime != nil {
		return *p.OneTime
	}
	return true
}

func resolveKey(cmd *cobra.Command) (key string, manual bool, err error) {
	if k, _ := cmd.Flags().GetString("key"); k != "" {
		return k, true, nil
	}
	k, err := crypto.GenerateKey()
	if err != nil {
		return "", false, fmt.Errorf("generate key: %w", err)
	}
	return k, false, nil
}

// resolveArgon2 uses the profile override when set, else asks the server via
// /config. A failed lookup falls back to the default (non-Argon2) derivation,
// which every yopass server accepts.
func resolveArgon2(ctx context.Context, client *api.Client, p config.Profile) bool {
	if p.Argon2 != nil {
		return *p.Argon2
	}
	if cfg, err := client.Config(ctx); err == nil {
		return cfg.Argon2
	}
	return false
}

func changedString(cmd *cobra.Command, name string) string {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetString(name)
		return v
	}
	return ""
}

func expirationLabel(seconds int32) string {
	switch seconds {
	case 3600:
		return "1h"
	case 86400:
		return "1d"
	case 604800:
		return "1w"
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}
