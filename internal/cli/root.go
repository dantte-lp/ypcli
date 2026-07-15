// Package cli wires the ypcli command tree, global flags, configuration and
// profile resolution, and the central error-to-exit-code mapping.
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BuildInfo carries release metadata injected into the binary at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// app holds process-wide state shared by the command tree.
type app struct {
	build BuildInfo
}

// settings are the effective values for one command invocation, resolved with
// precedence flag > env (YPCLI_*) > active profile > global defaults > built-in.
type settings struct {
	api      string
	url      string
	token    string // explicit token from --token / YPCLI_TOKEN
	timeout  time.Duration
	jsonMode bool
	verbose  bool
	profile  config.Profile
	log      *slog.Logger
}

// Execute builds and runs the command tree, returning the process exit code.
func Execute(ctx context.Context, build BuildInfo) int {
	a := &app{build: build}
	root := a.newRootCmd()
	root.SilenceErrors = true
	root.SilenceUsage = true

	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}

	code := exitCode(err)
	jsonMode := rootBool(root, "json")
	_ = output.New(jsonMode, os.Stdout, os.Stderr).Error(code, err.Error()) // best-effort
	if isUsageError(err) {
		_ = root.Usage()
	}
	return code
}

func (a *app) newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ypcli",
		Short: "Share text and files as end-to-end-encrypted one-time secrets via yopass",
		Long: "ypcli publishes text and files to a yopass server with client-side\n" +
			"OpenPGP encryption. Built for CI, agents and teams: bearer-token auth,\n" +
			"JSON output, strict exit codes and multiple server profiles.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	pf := root.PersistentFlags()
	pf.StringP("profile", "p", "", "configuration profile to use")
	pf.String("api", "", "yopass API base URL")
	pf.String("url", "", "yopass public URL (for share links)")
	pf.String("token", "", "bearer token for authenticated instances")
	pf.Duration("timeout", 30*time.Second, "request timeout")
	pf.Bool("json", false, "machine-readable JSON output")
	pf.BoolP("verbose", "v", false, "verbose logging to stderr")
	pf.String("config", "", "config file path (default: $XDG_CONFIG_HOME/ypcli/config.yaml)")

	root.AddCommand(
		a.newSendCmd(),
		a.newReceiveCmd(),
		a.newConfigCmd(),
		a.newVersionCmd(),
		a.newMCPCmd(),
	)
	return root
}

// resolve computes the effective settings for a command invocation.
func (a *app) resolve(cmd *cobra.Command) (*settings, error) {
	root := cmd.Root()

	cfgPath, err := configPath(root)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errConfig, err)
	}

	profName := stringFlagOrEnv(root, "profile", "YPCLI_PROFILE")
	profile, err := cfg.Effective(profName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errConfig, err)
	}

	v := viper.New()
	v.SetEnvPrefix("YPCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	if err := v.BindPFlags(root.PersistentFlags()); err != nil {
		return nil, fmt.Errorf("bind flags: %w", err)
	}
	// The active profile forms the default layer beneath flags and env.
	v.SetDefault("api", coalesce(profile.API, config.DefaultAPI))
	v.SetDefault("url", coalesce(profile.URL, config.DefaultURL))
	v.SetDefault("timeout", 30*time.Second)

	verbose := v.GetBool("verbose")
	return &settings{
		api:      strings.TrimSuffix(v.GetString("api"), "/"),
		url:      strings.TrimSuffix(v.GetString("url"), "/"),
		token:    v.GetString("token"),
		timeout:  v.GetDuration("timeout"),
		jsonMode: v.GetBool("json"),
		verbose:  verbose,
		profile:  profile,
		log:      newLogger(cmd.ErrOrStderr(), verbose),
	}, nil
}

// newLogger returns a slog logger writing to w. At non-verbose levels only
// warnings and above are emitted, keeping normal runs quiet.
func newLogger(w io.Writer, verbose bool) *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

// printer builds the output printer for a command from its streams.
func (s *settings) printer(cmd *cobra.Command) output.Printer {
	return output.New(s.jsonMode, cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func configPath(root *cobra.Command) (string, error) {
	if p := stringFlagOrEnv(root, "config", "YPCLI_CONFIG"); p != "" {
		return p, nil
	}
	p, err := config.DefaultPath()
	if err != nil {
		return "", fmt.Errorf("%w: %w", errConfig, err)
	}
	return p, nil
}

func stringFlagOrEnv(cmd *cobra.Command, flag, env string) string {
	if f := cmd.PersistentFlags().Lookup(flag); f != nil && f.Changed {
		return f.Value.String()
	}
	return os.Getenv(env)
}

func rootBool(root *cobra.Command, name string) bool {
	b, _ := root.PersistentFlags().GetBool(name)
	return b
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
