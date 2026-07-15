package cli

import (
	"fmt"
	"sort"

	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/spf13/cobra"
)

func (a *app) newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage server profiles",
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	cmd.AddCommand(
		a.newConfigAddCmd(),
		a.newConfigListCmd(),
		a.newConfigUseCmd(),
		a.newConfigRemoveCmd(),
		a.newConfigDefaultsCmd(),
	)
	return cmd
}

func (a *app) newConfigDefaultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defaults",
		Short: "Set global defaults applied beneath every profile",
		Long: "Set global defaults stored at the top level of the config file. They\n" +
			"fill any field the active profile, flags and env leave unset, so ypcli\n" +
			"can target a self-hosted server without creating a profile.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, cfg, err := loadConfigForWrite(cmd)
			if err != nil {
				return err
			}
			d := cfg.Defaults
			if v := changedString(cmd, "api"); v != "" {
				d.API = v
			}
			if v := changedString(cmd, "url"); v != "" {
				d.URL = v
			}
			if v := changedString(cmd, "expiration"); v != "" {
				d.Expiration = v
			}
			if v := changedString(cmd, "token-command"); v != "" {
				d.TokenCommand = v
			}
			applyVaultFlags(cmd, &d)
			cfg.Defaults = d
			if err := cfg.Save(path); err != nil {
				return fmt.Errorf("%w: %w", errConfig, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "saved global defaults")
			return nil
		},
	}
	f := cmd.Flags()
	f.String("api", "", "default yopass API base URL")
	f.String("url", "", "default yopass public URL")
	f.String("expiration", "", "default expiration (1h, 1d, 1w)")
	f.String("token-command", "", "default shell command that prints a bearer token")
	addVaultConfigFlags(cmd)
	return cmd
}

func (a *app) newConfigAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create or update a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, cfg, err := loadConfigForWrite(cmd)
			if err != nil {
				return err
			}
			p := cfg.Profiles[args[0]]
			if v := changedString(cmd, "api"); v != "" {
				p.API = v
			}
			if v := changedString(cmd, "url"); v != "" {
				p.URL = v
			}
			if v := changedString(cmd, "expiration"); v != "" {
				p.Expiration = v
			}
			if v := changedString(cmd, "token-command"); v != "" {
				p.TokenCommand = v
			}
			applyVaultFlags(cmd, &p)
			cfg.Profiles[args[0]] = p
			if cfg.Active == "" {
				cfg.Active = args[0]
			}
			if err := cfg.Save(path); err != nil {
				return fmt.Errorf("%w: %w", errConfig, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved profile %q\n", args[0])
			return nil
		},
	}
	f := cmd.Flags()
	f.String("api", "", "yopass API base URL")
	f.String("url", "", "yopass public URL")
	f.String("expiration", "", "default expiration (1h, 1d, 1w)")
	f.String("token-command", "", "shell command that prints a bearer token")
	addVaultConfigFlags(cmd)
	return cmd
}

// addVaultConfigFlags registers the profile-level Vault/OpenBao connection flags
// shared by `config add` and `config defaults`.
func addVaultConfigFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("vault-addr", "", "Vault/OpenBao address")
	f.String("vault-mount", "", "Vault/OpenBao KV v2 mount")
	f.String("vault-namespace", "", "Vault/OpenBao namespace")
	f.String("vault-token-command", "", "command that prints a Vault/OpenBao token")
}

// applyVaultFlags merges any set vault-* flags into the profile's vault block.
func applyVaultFlags(cmd *cobra.Command, p *config.Profile) {
	v := config.VaultConfig{}
	if p.Vault != nil {
		v = *p.Vault
	}
	set := false
	if x := changedString(cmd, "vault-addr"); x != "" {
		v.Addr = x
		set = true
	}
	if x := changedString(cmd, "vault-mount"); x != "" {
		v.Mount = x
		set = true
	}
	if x := changedString(cmd, "vault-namespace"); x != "" {
		v.Namespace = x
		set = true
	}
	if x := changedString(cmd, "vault-token-command"); x != "" {
		v.TokenCommand = x
		set = true
	}
	if set || p.Vault != nil {
		p.Vault = &v
	}
}

func (a *app) newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadConfigForWrite(cmd)
			if err != nil {
				return err
			}
			if cfg.Defaults.API != "" || cfg.Defaults.URL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  (defaults)\t%s\n", cfg.Defaults.API)
			}
			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				marker := "  "
				if name == cfg.Active {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\t%s\n", marker, name, cfg.Profiles[name].API)
			}
			return nil
		},
	}
}

func (a *app) newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, cfg, err := loadConfigForWrite(cmd)
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return usage("unknown profile %q", args[0])
			}
			cfg.Active = args[0]
			if err := cfg.Save(path); err != nil {
				return fmt.Errorf("%w: %w", errConfig, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "active profile: %s\n", args[0])
			return nil
		},
	}
}

func (a *app) newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, cfg, err := loadConfigForWrite(cmd)
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return usage("unknown profile %q", args[0])
			}
			delete(cfg.Profiles, args[0])
			if cfg.Active == args[0] {
				cfg.Active = ""
			}
			if err := cfg.Save(path); err != nil {
				return fmt.Errorf("%w: %w", errConfig, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed profile %q\n", args[0])
			return nil
		},
	}
}

func loadConfigForWrite(cmd *cobra.Command) (string, *config.Config, error) {
	path, err := configPath(cmd.Root())
	if err != nil {
		return "", nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", errConfig, err)
	}
	return path, cfg, nil
}
