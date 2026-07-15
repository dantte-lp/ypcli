// Package mcpserver builds an MCP server that exposes ypcli's send/receive
// operations as tools, so AI agents (Claude, Codex, Gemini, …) can share and
// fetch yopass secrets. It reuses internal/share, so behavior is identical to
// the CLI. Connection settings come from the ypcli config profiles on the host.
package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/dantte-lp/ypcli/internal/share"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures the MCP server.
type Options struct {
	ConfigPath string // path to the ypcli config file
	ReadOnly   bool   // omit receive_secret when true (send-only deployments)
	Version    string // client build version, reported by server_version
}

// New builds the MCP server and registers ypcli's tools.
func New(o Options) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "ypcli", Version: o.Version}, nil)
	r := &registry{o: o}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_secret",
		Description: "Encrypt and publish a text secret to yopass, returning a one-time share URL.",
	}, r.sendSecret)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_file",
		Description: "Encrypt and publish a file (by path) to yopass, returning a one-time share URL.",
	}, r.sendFile)
	if !o.ReadOnly {
		mcp.AddTool(s, &mcp.Tool{
			Name: "receive_secret",
			Description: "Fetch and decrypt a yopass secret by share URL or id+key. " +
				"NOTE: one-time secrets are consumed (deleted) on the first successful fetch.",
		}, r.receiveSecret)
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_profiles",
		Description: "List the configured ypcli server profiles.",
	}, r.listProfiles)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "server_version",
		Description: "Report the ypcli client version and the target yopass server version.",
	}, r.serverVersion)

	return s
}

type registry struct{ o Options }

// ---- tool inputs / outputs -------------------------------------------------

type sendSecretInput struct {
	Text        string `json:"text" jsonschema:"the secret text to share"`
	Profile     string `json:"profile,omitempty" jsonschema:"config profile to use (optional; default active)"`
	Expiration  string `json:"expiration,omitempty" jsonschema:"lifetime: 1h, 1d or 1w (default 1h)"`
	OneTime     *bool  `json:"one_time,omitempty" jsonschema:"delete after first view (default true)"`
	RequireAuth bool   `json:"require_auth,omitempty" jsonschema:"require authentication to view (server support needed)"`
}

type sendFileInput struct {
	Path       string `json:"path" jsonschema:"absolute path to the file to share"`
	Profile    string `json:"profile,omitempty" jsonschema:"config profile to use (optional; default active)"`
	Expiration string `json:"expiration,omitempty" jsonschema:"lifetime: 1h, 1d or 1w (default 1h)"`
	OneTime    *bool  `json:"one_time,omitempty" jsonschema:"delete after first view (default true)"`
}

type sendOutput struct {
	URL        string `json:"url" jsonschema:"the one-time share URL to hand to the recipient"`
	ID         string `json:"id"`
	Key        string `json:"key" jsonschema:"the decryption key (embedded in the URL unless a manual key was used)"`
	Expiration string `json:"expiration"`
	OneTime    bool   `json:"one_time"`
	File       bool   `json:"file"`
}

type receiveInput struct {
	URL     string `json:"url,omitempty" jsonschema:"a yopass share URL"`
	ID      string `json:"id,omitempty" jsonschema:"secret id (when no url is given)"`
	Key     string `json:"key,omitempty" jsonschema:"decryption key (required for id, or for manual-key URLs)"`
	File    bool   `json:"file,omitempty" jsonschema:"treat as a file secret (with id)"`
	Profile string `json:"profile,omitempty" jsonschema:"config profile to use (optional; default active)"`
}

type receiveOutput struct {
	Content       string `json:"content,omitempty" jsonschema:"the decrypted secret (for UTF-8 text)"`
	ContentBase64 string `json:"content_base64,omitempty" jsonschema:"base64 of the decrypted bytes (for binary payloads)"`
	Filename      string `json:"filename,omitempty" jsonschema:"embedded filename, for file secrets"`
	File          bool   `json:"file"`
}

type noInput struct{}

type profileInfo struct {
	Name   string `json:"name"`
	API    string `json:"api"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

type listProfilesOutput struct {
	Profiles []profileInfo `json:"profiles"`
}

type versionInput struct {
	Profile string `json:"profile,omitempty" jsonschema:"config profile to use (optional; default active)"`
}

type versionOutput struct {
	Client string `json:"client"`
	Server string `json:"server"`
}

// ---- handlers --------------------------------------------------------------

func (r *registry) sendSecret(ctx context.Context, _ *mcp.CallToolRequest, in sendSecretInput) (*mcp.CallToolResult, sendOutput, error) {
	client, publicURL, prof, err := r.clientFor(ctx, in.Profile)
	if err != nil {
		return nil, sendOutput{}, err
	}
	exp, err := expiration(in.Expiration, prof)
	if err != nil {
		return nil, sendOutput{}, err
	}
	res, err := share.SendText(ctx, client, publicURL, strings.NewReader(in.Text), share.Options{
		Expiration: exp, OneTime: oneTime(in.OneTime, prof), RequireAuth: in.RequireAuth, Argon2: prof.Argon2,
	})
	if err != nil {
		return nil, sendOutput{}, err
	}
	return nil, toSendOutput(res), nil
}

func (r *registry) sendFile(ctx context.Context, _ *mcp.CallToolRequest, in sendFileInput) (*mcp.CallToolResult, sendOutput, error) {
	client, publicURL, prof, err := r.clientFor(ctx, in.Profile)
	if err != nil {
		return nil, sendOutput{}, err
	}
	exp, err := expiration(in.Expiration, prof)
	if err != nil {
		return nil, sendOutput{}, err
	}
	res, err := share.SendFile(ctx, client, publicURL, in.Path, share.Options{
		Expiration: exp, OneTime: oneTime(in.OneTime, prof), Argon2: prof.Argon2,
	})
	if err != nil {
		return nil, sendOutput{}, err
	}
	return nil, toSendOutput(res), nil
}

func (r *registry) receiveSecret(ctx context.Context, _ *mcp.CallToolRequest, in receiveInput) (*mcp.CallToolResult, receiveOutput, error) {
	client, _, _, err := r.clientFor(ctx, in.Profile)
	if err != nil {
		return nil, receiveOutput{}, err
	}
	id, key, file, err := target(in)
	if err != nil {
		return nil, receiveOutput{}, err
	}
	res, err := share.Receive(ctx, client, share.Target{ID: id, Key: key, File: file})
	if err != nil {
		return nil, receiveOutput{}, err
	}
	out := receiveOutput{Filename: res.Filename, File: res.File}
	if utf8.ValidString(res.Content) {
		out.Content = res.Content
	} else {
		out.ContentBase64 = base64.StdEncoding.EncodeToString([]byte(res.Content))
	}
	return nil, out, nil
}

func (r *registry) listProfiles(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, listProfilesOutput, error) {
	cfg, err := config.Load(r.o.ConfigPath)
	if err != nil {
		return nil, listProfilesOutput{}, err
	}
	out := listProfilesOutput{Profiles: []profileInfo{}}
	for name, p := range cfg.Profiles {
		out.Profiles = append(out.Profiles, profileInfo{
			Name:   name,
			API:    firstNonEmpty(p.API, cfg.Defaults.API),
			URL:    firstNonEmpty(p.URL, cfg.Defaults.URL),
			Active: name == cfg.Active,
		})
	}
	return nil, out, nil
}

func (r *registry) serverVersion(ctx context.Context, _ *mcp.CallToolRequest, in versionInput) (*mcp.CallToolResult, versionOutput, error) {
	client, _, _, err := r.clientFor(ctx, in.Profile)
	if err != nil {
		return nil, versionOutput{}, err
	}
	server := "unknown"
	if v, verr := client.Version(ctx); verr == nil {
		server = v
	}
	return nil, versionOutput{Client: r.o.Version, Server: server}, nil
}

// ---- helpers ---------------------------------------------------------------

// clientFor loads the config, resolves the (effective) profile, sources its
// token, and builds an API client plus the public share URL.
func (r *registry) clientFor(ctx context.Context, profileName string) (*api.Client, string, config.Profile, error) {
	cfg, err := config.Load(r.o.ConfigPath)
	if err != nil {
		return nil, "", config.Profile{}, err
	}
	prof, err := cfg.Effective(profileName)
	if err != nil {
		return nil, "", config.Profile{}, err
	}
	token, err := config.ResolveToken(ctx, "", prof.TokenCommand)
	if err != nil {
		return nil, "", prof, err
	}
	apiBase := strings.TrimSuffix(firstNonEmpty(prof.API, config.DefaultAPI), "/")
	publicURL := strings.TrimSuffix(firstNonEmpty(prof.URL, config.DefaultURL), "/")

	opts := []api.Option{}
	if token != "" {
		opts = append(opts, api.WithToken(token))
	}
	return api.New(apiBase, opts...), publicURL, prof, nil
}

func expiration(label string, p config.Profile) (int32, error) {
	label = firstNonEmpty(label, p.Expiration, config.DefaultExpiration)
	seconds, ok := crypto.ExpirationSeconds(label)
	if !ok {
		return 0, fmt.Errorf("invalid expiration %q: use 1h, 1d or 1w", label)
	}
	return seconds, nil
}

func oneTime(in *bool, p config.Profile) bool {
	if in != nil {
		return *in
	}
	if p.OneTime != nil {
		return *p.OneTime
	}
	return true
}

func target(in receiveInput) (id, key string, file bool, err error) {
	if in.URL != "" {
		var keyOpt bool
		id, key, file, keyOpt, err = crypto.ParseURL(in.URL)
		if err != nil {
			return "", "", false, err
		}
		if keyOpt || key == "" {
			if in.Key == "" {
				return "", "", false, fmt.Errorf("this link needs a manual key: set key")
			}
			key = in.Key
		}
		return id, key, file, nil
	}
	if in.ID == "" {
		return "", "", false, fmt.Errorf("provide a url or an id")
	}
	if in.Key == "" {
		return "", "", false, fmt.Errorf("key is required with id")
	}
	return in.ID, in.Key, in.File, nil
}

func toSendOutput(res share.SendResult) sendOutput {
	return sendOutput{
		URL: res.URL, ID: res.ID, Key: res.Key,
		Expiration: res.Expiration, OneTime: res.OneTime, File: res.File,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
