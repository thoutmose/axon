package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thoutmose/axon/api/internal/notesync"
)

type vaultSetup struct {
	Source struct {
		Host       string `json:"host"`
		Dir        string `json:"dir"`
		Status     string `json:"status"`
		PairedHost string `json:"paired_host"`
		LastReport struct {
			Notes     int      `json:"notes"`
			Conflicts []string `json:"conflicts"`
			Error     string   `json:"error"`
		} `json:"last_report"`
	} `json:"source"`
	Setup struct {
		Code    string `json:"code"`
		Install string `json:"install"`
		Docker  string `json:"docker"`
	} `json:"setup"`
}

// pair calls the pairing endpoint as a server does: no session, no Origin.
func pair(e *env, code string, empty bool) (int, string) {
	e.t.Helper()
	b, _ := json.Marshal(map[string]any{"code": code, "host": "srv-secondbrain", "empty": empty})
	code2, out := e.client().do("POST", "/v1/vault/pair", b, "Origin", "")
	var res struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(out, &res)
	return code2, res.Token
}

// A vault server is named in Axon, pairs once with a short-lived code, reports
// after each sync, and an empty vault starts with folders and templates.
func TestVaultServer(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	// By default notes live in Axon only.
	if out := alice.mustDo(200, "GET", "/v1/vault/source", nil); string(out) != "{\"source\":null}\n" {
		t.Fatalf("default source = %s", out)
	}
	// The host and folder end up in shell commands: no quotes, full paths only.
	for _, bad := range []string{`{"host":"srv'x","dir":"~/v"}`, `{"host":"srv","dir":"vault"}`, `{"host":"srv","dir":"~/a/../../etc"}`, `{"host":"srv","dir":"~/$(reboot)"}`} {
		alice.mustDo(422, "PUT", "/v1/vault/source", []byte(bad))
	}
	var s vaultSetup
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/vault/source", []byte(`{"host":"srv-secondbrain","dir":"~/SecondBrain"}`)), &s)
	if s.Source.Status != "waiting" || len(s.Setup.Code) != 9 || !strings.Contains(s.Setup.Install, origin+"/api/v1/vault/install.sh") ||
		!strings.Contains(s.Setup.Install, "--code "+s.Setup.Code) || !strings.Contains(s.Setup.Docker, "axon-api:test") || !strings.Contains(s.Setup.Docker, `"$HOME"/'SecondBrain':/vault`) {
		t.Fatalf("setup = %+v", s)
	}
	bob.mustDo(200, "GET", "/v1/vault/source", nil) // Bob has none
	if out := bob.mustDo(200, "GET", "/v1/vault/source", nil); strings.Contains(string(out), "srv-secondbrain") {
		t.Fatal("Bob sees Alice's vault server")
	}

	// A wrong code is refused; the right one works once, typed loosely.
	if code, _ := pair(e, "AAAA-AAAA", true); code != 401 {
		t.Fatalf("wrong code = %d", code)
	}
	code, token := pair(e, strings.ToLower(strings.ReplaceAll(s.Setup.Code, "-", " ")), true)
	if code != 200 || !strings.HasPrefix(token, "axon_") {
		t.Fatalf("pair = %d %q", code, token)
	}
	if again, _ := pair(e, s.Setup.Code, true); again != 401 {
		t.Fatalf("second use of a code = %d", again)
	}
	// The vault was empty: Home and the templates are in Axon for the first sync.
	notes := alice.notes()
	if _, ok := notes["Home.md"]; !ok {
		t.Fatal("no Home note")
	}
	if _, ok := notes["Templates/permanent.md"]; !ok {
		t.Fatal("no templates")
	}
	var scaffold struct {
		Added   []string `json:"added"`
		Folders []string `json:"folders"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/vault/scaffold", nil), &scaffold)
	if len(scaffold.Added) != 0 || len(scaffold.Folders) == 0 {
		t.Fatalf("scaffold again = %+v", scaffold)
	}

	// The server reports with its token; Axon shows it as connected.
	server := e.client()
	if c, _ := server.do("POST", "/v1/vault/heartbeat", []byte(`{"host":"srv-secondbrain","notes":12,"conflicts":["a.md"]}`), "Authorization", "Bearer "+token); c != 204 {
		t.Fatalf("heartbeat = %d", c)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/vault/source", nil), &s)
	if s.Source.Status != "connected" || s.Source.PairedHost != "srv-secondbrain" || s.Source.LastReport.Notes != 12 || len(s.Source.LastReport.Conflicts) != 1 {
		t.Fatalf("after heartbeat = %+v", s.Source)
	}
	server.do("POST", "/v1/vault/heartbeat", []byte(`{"error":"disk full"}`), "Authorization", "Bearer "+token)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/vault/source", nil), &s)
	if s.Source.Status != "failing" {
		t.Fatalf("after a failed sync = %s", s.Source.Status)
	}
	// Bob's token cannot report for a vault he does not have.
	var bobTok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(bob.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"x","scope":"write","expires_in_days":1}`)), &bobTok)
	if c, _ := server.do("POST", "/v1/vault/heartbeat", []byte(`{}`), "Authorization", "Bearer "+bobTok.Token); c != 404 {
		t.Fatalf("heartbeat without a vault = %d", c)
	}

	// Pairing again (a new server, say) replaces the token.
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/vault/source", []byte(`{"host":"srv-new","dir":"/srv/vault","scaffold":false}`)), &s)
	_, token2 := pair(e, s.Setup.Code, true)
	if c, _ := server.do("GET", "/v1/notes/sync/version", nil, "Authorization", "Bearer "+token); c != 401 {
		t.Fatalf("old token after pairing again = %d", c)
	}
	// Removing the vault server revokes its token and keeps the notes.
	alice.mustDo(204, "DELETE", "/v1/vault/source", nil)
	alice.mustDo(404, "DELETE", "/v1/vault/source", nil)
	if c, _ := server.do("GET", "/v1/notes/sync/version", nil, "Authorization", "Bearer "+token2); c != 401 {
		t.Fatalf("token after removing the vault server = %d", c)
	}
	if _, ok := alice.notes()["Home.md"]; !ok {
		t.Fatal("removing the vault server deleted notes")
	}

	// The install script and the program are public.
	if out := e.client().mustDo(200, "GET", "/v1/vault/install.sh", nil); !strings.Contains(string(out), "base='"+origin+"'") {
		t.Fatalf("install.sh = %.200s", out)
	}
	e.client().mustDo(404, "GET", "/v1/vault/agent?os=plan9&arch=mips", nil)
	if out := e.client().mustDo(200, "GET", "/v1/vault/agent?os="+runtime.GOOS+"&arch="+runtime.GOARCH, nil); len(out) < 1000 {
		t.Fatalf("agent is %d bytes", len(out))
	}
}

// notes-sync -pair on an empty folder: it pairs, saves the token for its
// owner only, makes the folders, and the first sync writes the templates.
func TestVaultPairingFillsAnEmptyVault(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Ideas/first.md","content":"# First"}`))
	var s vaultSetup
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/vault/source", []byte(`{"host":"srv","dir":"~/vault"}`)), &s)

	proxy := httptest.NewServer(http.StripPrefix("/api", e.srv.Config.Handler))
	t.Cleanup(proxy.Close)
	dir, conf := t.TempDir(), t.TempDir()
	cfg := notesync.Config{Dir: dir, URL: proxy.URL, TwoWay: true, StatePath: filepath.Join(conf, "state.json")}
	tokenFile := filepath.Join(conf, "axon", "vault-token")
	p, err := notesync.Pair(context.Background(), &cfg, s.Setup.Code, tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(tokenFile); err != nil || info.Mode().Perm() != 0o600 || len(p.Folders) == 0 {
		t.Fatalf("token file %v %v, folders %v", info, err, p.Folders)
	}
	for _, f := range p.Folders {
		if info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil || !info.IsDir() {
			t.Fatalf("folder %s not made: %v", f, err)
		}
	}
	rep, err := notesync.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"Home.md", "Templates/Fleeting note.md", "Ideas/first.md"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Fatalf("%s not written: %v", f, err)
		}
	}
	if err := notesync.Heartbeat(context.Background(), cfg, rep, nil); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/vault/source", nil), &s)
	if s.Source.Status != "connected" || s.Source.LastReport.Notes != rep.Notes || rep.Notes < 3 {
		t.Fatalf("status %s, report %+v, run %+v", s.Source.Status, s.Source.LastReport, rep)
	}

	// A vault that already has notes is left as it is.
	full := t.TempDir()
	_ = os.WriteFile(filepath.Join(full, "mine.md"), []byte("# Mine"), 0o600)
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/vault/source", []byte(`{"host":"srv","dir":"~/vault"}`)), &s)
	cfg2 := notesync.Config{Dir: full, URL: proxy.URL}
	p2, err := notesync.Pair(context.Background(), &cfg2, s.Setup.Code, filepath.Join(t.TempDir(), "tok"))
	if err != nil || len(p2.Folders) != 0 || len(p2.Scaffold) != 0 {
		t.Fatalf("pairing a full vault: %+v %v", p2, err)
	}
}
