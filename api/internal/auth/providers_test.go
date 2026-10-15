package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestProvidersFromEnv(t *testing.T) {
	if ps, err := ProvidersFromEnv(env(nil)); err != nil || len(ps) != 0 {
		t.Fatalf("none configured: %v %v", ps, err)
	}
	ps, err := ProvidersFromEnv(env(map[string]string{
		"AXON_GOOGLE_CLIENT_ID": "g", "AXON_GOOGLE_CLIENT_SECRET": "gs",
		"AXON_MICROSOFT_CLIENT_ID": "m", "AXON_MICROSOFT_CLIENT_SECRET": "ms",
		"AXON_DISCORD_CLIENT_ID": "d", "AXON_DISCORD_CLIENT_SECRET": "ds",
		"AXON_SLACK_CLIENT_ID": "s", // no secret: off
	}))
	if err != nil || len(ps) != 3 || ps[0].ID != "google" || ps[1].ID != "microsoft" || ps[2].UserURL == "" {
		t.Fatalf("providers = %+v, %v", ps, err)
	}
	if _, err := ProvidersFromEnv(env(map[string]string{"AXON_MICROSOFT_CLIENT_ID": "m", "AXON_MICROSOFT_CLIENT_SECRET": "ms", "AXON_MICROSOFT_TENANT": "a/../b"})); err == nil {
		t.Fatal("a tenant with a path was accepted")
	}
	if _, err := ProvidersFromEnv(env(map[string]string{"AXON_APPLE_CLIENT_ID": "a", "AXON_APPLE_PRIVATE_KEY": "nope"})); err == nil {
		t.Fatal("a bad Apple key was accepted")
	}
	if _, err := ProvidersFromEnv(env(map[string]string{"AXON_OIDC_ISSUER": "http://idp.example"})); err == nil {
		t.Fatal("OIDC without a client was accepted")
	}
}

// Apple's client secret is an ES256 JWT its servers can verify.
func TestAppleClientSecret(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	ps, err := ProvidersFromEnv(env(map[string]string{"AXON_APPLE_CLIENT_ID": "com.example.axon", "AXON_APPLE_TEAM_ID": "TEAM123", "AXON_APPLE_KEY_ID": "KEY456",
		"AXON_APPLE_PRIVATE_KEY": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))}))
	if err != nil || len(ps) != 1 {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	jwt, err := ps[0].clientSecret(now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(jwt, ".")
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if len(sig) != 64 || !ecdsa.Verify(&key.PublicKey, digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("the signature does not verify")
	}
	var head, body map[string]any
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	b, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(h, &head)
	_ = json.Unmarshal(b, &body)
	if head["alg"] != "ES256" || head["kid"] != "KEY456" || body["iss"] != "TEAM123" || body["sub"] != "com.example.axon" ||
		body["aud"] != "https://appleid.apple.com" || body["exp"].(float64) != float64(now.Add(5*time.Minute).Unix()) {
		t.Fatalf("head %v body %v", head, body)
	}
}

func TestIDTokenChecks(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	google := Provider{ID: "google", ClientID: "cid", Issuers: []string{"https://accounts.google.com"}}
	ms, _ := ProvidersFromEnv(env(map[string]string{"AXON_MICROSOFT_CLIENT_ID": "cid", "AXON_MICROSOFT_CLIENT_SECRET": "x"}))
	ok := claims{Iss: "https://accounts.google.com", Sub: "1", Aud: json.RawMessage(`"cid"`), Exp: now.Unix() + 60, Nonce: "n", Email: "a@example.org", Name: "Ada"}
	if id, err := ok.check(google, "n", now); err != nil || id.Subject != "1" || id.Name != "Ada" {
		t.Fatalf("valid token: %+v %v", id, err)
	}
	bad := map[string]func(c *claims){
		"issuer":   func(c *claims) { c.Iss = "https://evil.example" },
		"audience": func(c *claims) { c.Aud = json.RawMessage(`"other"`) },
		"azp":      func(c *claims) { c.Aud = json.RawMessage(`["cid","other"]`); c.Azp = "other" },
		"expired":  func(c *claims) { c.Exp = now.Unix() - 1 },
		"nonce":    func(c *claims) { c.Nonce = "replayed" },
		"subject":  func(c *claims) { c.Sub = "" },
	}
	for name, change := range bad {
		c := ok
		change(&c)
		if _, err := c.check(google, "n", now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	c := ok
	c.Aud = json.RawMessage(`["cid","other"]`)
	c.Azp = "cid"
	if _, err := c.check(google, "n", now); err != nil {
		t.Fatalf("several audiences with azp: %v", err)
	}
	// Microsoft's issuer names the tenant.
	c = ok
	c.Iss = "https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"
	if _, err := c.check(ms[0], "n", now); err != nil {
		t.Fatalf("microsoft: %v", err)
	}
	c.Iss = "https://login.microsoftonline.com.evil.example/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"
	if _, err := c.check(ms[0], "n", now); err == nil {
		t.Fatal("a look-alike Microsoft issuer was accepted")
	}
	if safeNext("//evil.example") != "/" || safeNext("https://evil.example") != "/" || safeNext("/\\evil") != "/" || safeNext("/tasks?x=1") != "/tasks?x=1" {
		t.Fatal("safeNext lets a redirect leave the site")
	}
}
