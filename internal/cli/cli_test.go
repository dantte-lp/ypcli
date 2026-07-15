package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dantte-lp/ypcli/internal/crypto"
)

// fakeServer is a minimal in-memory yopass API for command tests.
type fakeServer struct {
	mu      sync.Mutex
	secrets map[string]string
	files   map[string][]byte
	argon2  bool
}

func newFakeServer() *fakeServer {
	return &fakeServer{secrets: map[string]string{}, files: map[string][]byte{}}
}

func (f *fakeServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ARGON2": f.argon2})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": "test-1.0"})
	})
	mux.HandleFunc("/create/secret", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message string `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		id, _ := crypto.GenerateID()
		f.mu.Lock()
		f.secrets[id] = body.Message
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": id})
	})
	mux.HandleFunc("/create/file", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		id, _ := crypto.GenerateID()
		f.mu.Lock()
		f.files[id] = data
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": id})
	})
	mux.HandleFunc("/secret/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/secret/")
		f.mu.Lock()
		msg, ok := f.secrets[id]
		delete(f.secrets, id)
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": msg})
	})
	mux.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/file/")
		f.mu.Lock()
		data, ok := f.files[id]
		delete(f.files, id)
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(data)
	})
	return mux
}

// run executes the ypcli command tree with args, isolated config, and captured
// output, returning stdout, stderr and the exit code.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := &app{build: BuildInfo{Version: "test", Commit: "abc", Date: "today"}}
	root := a.newRootCmd()
	root.SilenceErrors = true
	root.SilenceUsage = true

	var out, errw bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errw)
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())
	code := 0
	if err != nil {
		code = exitCode(err)
	}
	return out.String(), errw.String(), code
}

func TestSendReceiveTextRoundTrip(t *testing.T) {
	fs := newFakeServer()
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()

	// send (json to capture URL)
	out, _, code := runStdin(t, "top secret", "--api", srv.URL, "--url", srv.URL, "--json", "send")
	if code != 0 {
		t.Fatalf("send exit = %d, stderr", code)
	}
	var res struct{ URL string }
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("send output not json: %v (%q)", err, out)
	}

	// receive
	rout, _, rcode := run(t, "receive", res.URL, "--api", srv.URL)
	if rcode != 0 {
		t.Fatalf("receive exit = %d", rcode)
	}
	if rout != "top secret" {
		t.Errorf("receive stdout = %q, want plaintext", rout)
	}
}

func TestSendFileReceiveToDir(t *testing.T) {
	fs := newFakeServer()
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()

	dir := t.TempDir()
	srcFile := filepath.Join(dir, "creds.env")
	if err := os.WriteFile(srcFile, []byte("USER=admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, _, code := run(t, "send", "--file", srcFile, "--api", srv.URL, "--url", srv.URL, "--json")
	if code != 0 {
		t.Fatalf("send exit = %d", code)
	}
	var res struct {
		URL  string `json:"url"`
		File bool   `json:"file"`
	}
	json.Unmarshal([]byte(out), &res)
	if !res.File || !strings.Contains(res.URL, "/#/f/") {
		t.Fatalf("expected file URL, got %+v", res)
	}

	outDir := t.TempDir()
	_, _, rcode := run(t, "receive", res.URL, "--api", srv.URL, "-o", outDir)
	if rcode != 0 {
		t.Fatalf("receive exit = %d", rcode)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "creds.env"))
	if err != nil {
		t.Fatalf("output file: %v", err)
	}
	if string(got) != "USER=admin\n" {
		t.Errorf("decrypted file = %q", got)
	}
}

// Regression: receiving a file into a not-yet-existing directory (trailing
// separator) must create it and write under the embedded filename.
func TestReceiveFileToNonexistentDir(t *testing.T) {
	fs := newFakeServer()
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()

	dir := t.TempDir()
	src := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(src, []byte("DATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, code := run(t, "send", "--file", src, "--api", srv.URL, "--url", srv.URL, "--json")
	if code != 0 {
		t.Fatalf("send exit = %d", code)
	}
	var res struct{ URL string }
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "new", "sub") + string(os.PathSeparator)
	_, _, rcode := run(t, "receive", res.URL, "--api", srv.URL, "-o", target)
	if rcode != 0 {
		t.Fatalf("receive exit = %d", rcode)
	}
	got, err := os.ReadFile(filepath.Join(target, "payload.bin"))
	if err != nil {
		t.Fatalf("expected file created in new dir: %v", err)
	}
	if string(got) != "DATA" {
		t.Errorf("content = %q", got)
	}
}

func TestReceiveConsumedOneTimeExit6(t *testing.T) {
	fs := newFakeServer()
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()

	out, _, _ := runStdin(t, "x", "--api", srv.URL, "--url", srv.URL, "--json", "send")
	var res struct{ URL string }
	json.Unmarshal([]byte(out), &res)

	// first receive consumes it
	run(t, "receive", res.URL, "--api", srv.URL)
	// second receive should be not-found -> exit 6
	_, _, code := run(t, "receive", res.URL, "--api", srv.URL)
	if code != 6 {
		t.Errorf("second receive exit = %d, want 6", code)
	}
}

func TestManualKeyURLRequiresKey(t *testing.T) {
	_, _, code := run(t, "receive", "https://y/#/c/someid", "--api", "https://y")
	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage)", code)
	}
}

func TestUnknownCommandExit(t *testing.T) {
	_, _, code := run(t, "frobnicate")
	if code == 0 {
		t.Error("unknown command should be non-zero exit")
	}
}

func TestConfigLifecycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)

	mustRun := func(args ...string) string {
		a := &app{}
		root := a.newRootCmd()
		root.SilenceErrors, root.SilenceUsage = true, true
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return out.String()
	}

	mustRun("config", "add", "work", "--api", "https://api.corp", "--url", "https://yp.corp")
	list := mustRun("config", "list")
	if !strings.Contains(list, "work") || !strings.Contains(list, "https://api.corp") {
		t.Errorf("list = %q", list)
	}
	if !strings.Contains(list, "* work") {
		t.Errorf("first profile should be active: %q", list)
	}
	mustRun("config", "remove", "work")
	list = mustRun("config", "list")
	if strings.Contains(list, "work") {
		t.Errorf("profile not removed: %q", list)
	}
}

func TestVersionJSON(t *testing.T) {
	fs := newFakeServer()
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()

	out, _, code := run(t, "version", "--api", srv.URL, "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var v struct {
		Version string `json:"version"`
		Server  string `json:"server"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("not json: %v (%q)", err, out)
	}
	if v.Version != "test" || v.Server != "test-1.0" {
		t.Errorf("version json = %+v", v)
	}
}

func TestSendFromVault(t *testing.T) {
	// Fake Vault/OpenBao KV v2 endpoint.
	vaultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "vt" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path != "/v1/secret/data/db" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": map[string]any{"password": "hunter2"}},
		})
	}))
	defer vaultSrv.Close()

	fs := newFakeServer()
	yp := httptest.NewServer(fs.handler())
	defer yp.Close()

	out, _, code := run(t, "send", "--api", yp.URL, "--url", yp.URL, "--json",
		"--vault-addr", vaultSrv.URL, "--vault-token", "vt",
		"--vault-path", "db", "--vault-field", "password")
	if code != 0 {
		t.Fatalf("send from vault exit = %d", code)
	}
	var res struct{ URL string }
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not json: %v (%q)", err, out)
	}
	got, _, rcode := run(t, "receive", res.URL, "--api", yp.URL)
	if rcode != 0 {
		t.Fatalf("receive exit = %d", rcode)
	}
	if got != "hunter2" {
		t.Errorf("decrypted vault secret = %q, want hunter2", got)
	}
}

func TestSendFromVaultMissingFieldFlag(t *testing.T) {
	_, _, code := run(t, "send", "--vault-path", "db", "--vault-addr", "http://x", "--vault-token", "t")
	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage: --vault-field required)", code)
	}
}

// runStdin is like run but feeds stdinData to the command as piped stdin.
func runStdin(t *testing.T, stdinData string, argsThenSubcmdLast ...string) (string, string, int) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Reorder: the caller passes flags... plus the subcommand as the final arg
	// for readability; move the subcommand to the front.
	sub := argsThenSubcmdLast[len(argsThenSubcmdLast)-1]
	rest := argsThenSubcmdLast[:len(argsThenSubcmdLast)-1]
	args := append([]string{sub}, rest...)

	a := &app{build: BuildInfo{Version: "test"}}
	root := a.newRootCmd()
	root.SilenceErrors, root.SilenceUsage = true, true

	var out, errw bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errw)
	root.SetIn(pipeStdin(t, stdinData))
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())
	code := 0
	if err != nil {
		code = exitCode(err)
	}
	return out.String(), errw.String(), code
}

// pipeStdin returns an *os.File whose contents are stdinData, so the send
// command's char-device check treats it as piped input.
func pipeStdin(t *testing.T, data string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(data); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
