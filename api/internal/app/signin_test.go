package app

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/auth"
)

// fakeIdP plays an OpenID provider's token endpoint and an OAuth 2 user
// endpoint: Axon's server-side calls are routed to it.
type fakeIdP struct {
	mu      sync.Mutex
	subject string
	nonce   string // what the ID token says; "" copies the authorize request's
	iss     string
	forms   []url.Values
}

func (f *fakeIdP) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	f.mu.Lock()
	defer f.mu.Unlock()
	switch req.URL.Path {
	case "/token":
		b, _ := io.ReadAll(req.Body)
		form, _ := url.ParseQuery(string(b))
		f.forms = append(f.forms, form)
		claims, _ := json.Marshal(map[string]any{"iss": f.iss, "sub": f.subject, "aud": "client-1", "exp": time.Now().Add(time.Minute).Unix(),
			"nonce": f.nonce, "email": "ada@example.org", "name": "Ada Lovelace"})
		token := "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".c2ln"
		_ = json.NewEncoder(rec).Encode(map[string]string{"access_token": "at", "id_token": token})
	case "/users/@me":
		_ = json.NewEncoder(rec).Encode(map[string]string{"id": f.subject, "username": "ada", "global_name": "Ada"})
	default:
		rec.WriteHeader(404)
	}
	res := rec.Result()
	res.Request = req
	return res, nil
}

// browse makes a request without following redirects, checked against the
// contract, and returns the Location.
func (c *client) browse(path string, want int) string {
	c.e.t.Helper()
	req, _ := http.NewRequest("GET", c.e.srv.URL+path, nil)
	hc := &http.Client{Jar: c.http.Jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := hc.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	checkContract(c.e.t, req, nil, res, body)
	if res.StatusCode != want {
		c.e.t.Fatalf("GET %s = %d, want %d: %s", path, res.StatusCode, want, body)
	}
	return res.Header.Get("Location")
}

// Signing in with another account: the account's owner links it first; it
// is never matched by email; PKCE, state, and nonce are checked.
func TestSignInWithAnotherAccount(t *testing.T) {
	idp := &fakeIdP{subject: "g-123", iss: "https://idp.example"}
	old := auth.OAuthClient
	auth.OAuthClient = &http.Client{Transport: idp}
	t.Cleanup(func() { auth.OAuthClient = old })
	e := newEnvWith(t, auth.Config{Providers: []auth.Provider{
		{ID: "google", Name: "Google", ClientID: "client-1", Secret: "s3cret", AuthURL: "https://idp.example/authorize", TokenURL: "https://idp.example/token", //nolint:gosec // a test provider
			Scopes: []string{"openid", "email", "profile"}, Issuers: []string{"https://idp.example"}},
		{ID: "discord", Name: "Discord", ClientID: "client-1", Secret: "s3cret", AuthURL: "https://idp.example/authorize", TokenURL: "https://idp.example/token", //nolint:gosec // a test provider
			UserURL: "https://idp.example/users/@me", Scopes: []string{"identify"}},
	}})
	alice := e.client()
	alice.register("Alice")

	var providers struct {
		Items []struct{ ID, Name string } `json:"items"`
	}
	_ = json.Unmarshal(e.client().mustDo(200, "GET", "/v1/auth/providers", nil), &providers)
	if len(providers.Items) != 2 || providers.Items[0].Name != "Google" {
		t.Fatalf("providers = %+v", providers)
	}

	// start → the provider, with PKCE, a state, and a nonce.
	start := func(c *client, query string) url.Values {
		loc, _ := url.Parse(c.browse("/v1/auth/oauth/google/start"+query, 302))
		q := loc.Query()
		if loc.Host != "idp.example" || q.Get("client_id") != "client-1" || q.Get("redirect_uri") != origin+"/api/v1/auth/oauth/google/callback" ||
			q.Get("code_challenge_method") != "S256" || len(q.Get("code_challenge")) != 43 || q.Get("state") == "" || q.Get("nonce") == "" {
			t.Fatalf("authorize = %s", loc)
		}
		return q
	}
	e.client().browse("/v1/auth/oauth/github/start", 404)
	e.client().browse("/v1/auth/oauth/google/start?intent=link", 401)

	// Not linked, registration closed: no account is made, none is matched.
	stranger := e.client()
	q := start(stranger, "")
	idp.nonce = q.Get("nonce")
	if loc := stranger.browse("/v1/auth/oauth/google/callback?code=c1&state="+q.Get("state"), 302); loc != "/signin?error=not_linked" {
		t.Fatalf("unlinked sign-in went to %s", loc)
	}
	stranger.mustDo(401, "GET", "/v1/me", nil)
	if form := idp.forms[len(idp.forms)-1]; form.Get("code_verifier") == "" || form.Get("client_secret") != "s3cret" || form.Get("code") != "c1" {
		t.Fatalf("token request = %v", form)
	}

	// Alice links Google while signed in.
	q = start(alice, "?intent=link")
	idp.nonce = q.Get("nonce")
	if loc := alice.browse("/v1/auth/oauth/google/callback?code=c2&state="+q.Get("state"), 302); loc != "/account/security?linked=google" {
		t.Fatalf("link went to %s", loc)
	}
	var ids struct {
		Items []struct {
			ID, Provider, Email string
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/identities", nil), &ids)
	if len(ids.Items) != 1 || ids.Items[0].Provider != "google" || ids.Items[0].Email != "ada@example.org" {
		t.Fatalf("identities = %+v", ids)
	}

	// Now a new browser signs in as Alice with Google, and lands where it asked.
	phone := e.client()
	q = start(phone, "?next=/tasks")
	idp.nonce = q.Get("nonce")
	if loc := phone.browse("/v1/auth/oauth/google/callback?code=c3&state="+q.Get("state"), 302); loc != "/tasks" {
		t.Fatalf("sign-in went to %s", loc)
	}
	if out := phone.mustDo(200, "GET", "/v1/me", nil); !strings.Contains(string(out), `"display_name":"Alice"`) {
		t.Fatalf("signed in as %s", out)
	}
	if out := phone.mustDo(200, "GET", "/v1/me/audit", nil); !strings.Contains(string(out), "signin.provider") {
		t.Fatal("no audit event")
	}

	// A replayed or foreign state, a wrong nonce, another site's next: refused.
	q = start(phone, "?next=//evil.example")
	idp.nonce = q.Get("nonce")
	if loc := e.client().browse("/v1/auth/oauth/google/callback?code=c4&state="+q.Get("state"), 302); loc != "/signin?error=state" {
		t.Fatalf("another browser's state went to %s", loc)
	}
	if loc := phone.browse("/v1/auth/oauth/google/callback?code=c4&state="+q.Get("state"), 302); loc != "/" {
		t.Fatalf("an unsafe next went to %s", loc)
	}
	if loc := phone.browse("/v1/auth/oauth/google/callback?code=c4&state="+q.Get("state"), 302); loc != "/signin?error=state" {
		t.Fatalf("a used state went to %s", loc) // the cookie is gone
	}
	q = start(phone, "")
	idp.nonce = "someone-else"
	if loc := phone.browse("/v1/auth/oauth/google/callback?code=c5&state="+q.Get("state"), 302); loc != "/signin?error=provider" {
		t.Fatalf("a wrong nonce went to %s", loc)
	}
	q = start(phone, "")
	idp.nonce, idp.iss = q.Get("nonce"), "https://evil.example"
	if loc := phone.browse("/v1/auth/oauth/google/callback?code=c6&state="+q.Get("state"), 302); loc != "/signin?error=provider" {
		t.Fatalf("a wrong issuer went to %s", loc)
	}
	idp.iss = "https://idp.example"
	q = start(phone, "")
	if loc := phone.browse("/v1/auth/oauth/google/callback?error=access_denied&state="+q.Get("state"), 302); loc != "/signin?error=cancelled" {
		t.Fatalf("a cancelled sign-in went to %s", loc)
	}

	// Linking the same account again changes nothing.
	q = start(alice, "?intent=link")
	idp.nonce = q.Get("nonce")
	if loc := alice.browse("/v1/auth/oauth/google/callback?code=c7&state="+q.Get("state"), 302); loc != "/account/security?linked=google" {
		t.Fatalf("linking again went to %s", loc)
	}

	// Discord is OAuth 2: Axon asks who signed in.
	idp.subject = "d-1"
	loc, _ := url.Parse(alice.browse("/v1/auth/oauth/discord/start?intent=link", 302))
	if loc.Query().Get("nonce") != "" || loc.Query().Get("scope") != "identify" {
		t.Fatalf("discord authorize = %s", loc)
	}
	if to := alice.browse("/v1/auth/oauth/discord/callback?code=c8&state="+loc.Query().Get("state"), 302); to != "/account/security?linked=discord" {
		t.Fatalf("discord link went to %s", to)
	}

	// Unlinking; a person with no passkey cannot remove their last way in.
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/identities", nil), &ids)
	if len(ids.Items) != 2 {
		t.Fatalf("identities = %+v", ids)
	}
	alice.mustDo(204, "DELETE", "/v1/me/identities/"+ids.Items[1].ID, nil)
	alice.mustDo(404, "DELETE", "/v1/me/identities/"+ids.Items[1].ID, nil)
}

// With open registration, a new person signs up with another account and
// is sent to add a passkey; the account is theirs alone.
func TestSignUpWithAnotherAccount(t *testing.T) {
	idp := &fakeIdP{subject: "new-1", iss: "https://idp.example"}
	old := auth.OAuthClient
	auth.OAuthClient = &http.Client{Transport: idp}
	t.Cleanup(func() { auth.OAuthClient = old })
	e := newEnvWith(t, auth.Config{OpenRegistration: true, Providers: []auth.Provider{
		{ID: "slack", Name: "Slack", ClientID: "client-1", Secret: "s", AuthURL: "https://idp.example/authorize", TokenURL: "https://idp.example/token", //nolint:gosec // a test provider
			Scopes: []string{"openid"}, Issuers: []string{"https://idp.example"}},
	}})
	c := e.client()
	loc, _ := url.Parse(c.browse("/v1/auth/oauth/slack/start", 302))
	idp.nonce = loc.Query().Get("nonce")
	if to := c.browse("/v1/auth/oauth/slack/callback?code=x&state="+loc.Query().Get("state"), 302); to != "/account/security?welcome=slack" {
		t.Fatalf("sign-up went to %s", to)
	}
	if out := c.mustDo(200, "GET", "/v1/me", nil); !strings.Contains(string(out), `"display_name":"Ada Lovelace"`) {
		t.Fatalf("me = %s", out)
	}
	// Someone else cannot link an account that is already someone's.
	bob := e.client()
	bob.register("Bob")
	loc, _ = url.Parse(bob.browse("/v1/auth/oauth/slack/start?intent=link", 302))
	idp.nonce = loc.Query().Get("nonce")
	if to := bob.browse("/v1/auth/oauth/slack/callback?code=y&state="+loc.Query().Get("state"), 302); to != "/account/security?error=taken" {
		t.Fatalf("linking someone else's account went to %s", to)
	}
	if out := bob.mustDo(200, "GET", "/v1/me/identities", nil); string(out) != "{\"items\":[]}\n" {
		t.Fatalf("bob's identities = %s", out)
	}
	// The only way in cannot be removed.
	var ids struct {
		Items []struct{ ID string } `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me/identities", nil), &ids)
	c.mustDo(409, "DELETE", "/v1/me/identities/"+ids.Items[0].ID, nil)
}
