package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// Provider is a service people can sign in with (EP-109). Google, Microsoft,
// Apple, Slack, and generic providers speak OpenID Connect; Discord speaks
// OAuth 2 and is asked who the person is.
type Provider struct {
	ID       string // google, microsoft, apple, discord, slack, oidc
	Name     string // shown on the button
	ClientID string
	Secret   string
	AuthURL  string
	TokenURL string
	UserURL  string   // OAuth 2 only: the endpoint that says who signed in
	Scopes   []string // none for Apple (then it answers with a plain redirect)
	Issuers  []string // accepted "iss" values; a pattern for Microsoft's tenants
	issuerRe *regexp.Regexp
	// Apple signs its client secret as a short-lived JWT with a key.
	appleTeam, appleKeyID string
	appleKey              *ecdsa.PrivateKey
}

// OAuthClient talks to the providers; tests replace it.
var OAuthClient = &http.Client{Timeout: 15 * time.Second}

// ProvidersFromEnv reads the providers the operator configured. A provider
// is on when its client id and secret (or Apple's key) are set.
func ProvidersFromEnv(getenv func(string) string) ([]Provider, error) {
	var out []Provider
	pair := func(prefix string) (string, string, bool) {
		id, secret := getenv("AXON_"+prefix+"_CLIENT_ID"), getenv("AXON_"+prefix+"_CLIENT_SECRET")
		return id, secret, id != "" && secret != ""
	}
	if id, secret, ok := pair("GOOGLE"); ok {
		out = append(out, Provider{ID: "google", Name: "Google", ClientID: id, Secret: secret, //nolint:gosec // endpoints, not credentials
			AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
			Scopes: []string{"openid", "email", "profile"}, Issuers: []string{"https://accounts.google.com", "accounts.google.com"}})
	}
	if id, secret, ok := pair("MICROSOFT"); ok {
		tenant := getenv("AXON_MICROSOFT_TENANT")
		if tenant == "" {
			tenant = "common"
		}
		if !regexp.MustCompile(`^[A-Za-z0-9.-]{1,100}$`).MatchString(tenant) {
			return nil, errors.New("AXON_MICROSOFT_TENANT is a tenant id or domain, or common, organizations, consumers")
		}
		out = append(out, Provider{ID: "microsoft", Name: "Microsoft", ClientID: id, Secret: secret, //nolint:gosec // endpoints, not credentials
			AuthURL:  "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token",
			Scopes:   []string{"openid", "email", "profile"},
			issuerRe: regexp.MustCompile(`^https://login\.microsoftonline\.com/[0-9a-f-]{36}/v2\.0$`)})
	}
	if id, secret, ok := pair("DISCORD"); ok {
		out = append(out, Provider{ID: "discord", Name: "Discord", ClientID: id, Secret: secret, //nolint:gosec // endpoints, not credentials
			AuthURL: "https://discord.com/oauth2/authorize", TokenURL: "https://discord.com/api/oauth2/token",
			UserURL: "https://discord.com/api/users/@me", Scopes: []string{"identify"}})
	}
	if id, secret, ok := pair("SLACK"); ok {
		out = append(out, Provider{ID: "slack", Name: "Slack", ClientID: id, Secret: secret, //nolint:gosec // endpoints, not credentials
			AuthURL: "https://slack.com/openid/connect/authorize", TokenURL: "https://slack.com/api/openid.connect.token",
			Scopes: []string{"openid", "email", "profile"}, Issuers: []string{"https://slack.com"}})
	}
	if id := getenv("AXON_APPLE_CLIENT_ID"); id != "" {
		keyPEM := getenv("AXON_APPLE_PRIVATE_KEY")
		if f := getenv("AXON_APPLE_PRIVATE_KEY_FILE"); f != "" {
			b, err := os.ReadFile(f) //nolint:gosec // the operator's own setting
			if err != nil {
				return nil, fmt.Errorf("AXON_APPLE_PRIVATE_KEY_FILE: %w", err)
			}
			keyPEM = string(b)
		}
		key, err := parseAppleKey(keyPEM)
		if err != nil {
			return nil, err
		}
		team, kid := getenv("AXON_APPLE_TEAM_ID"), getenv("AXON_APPLE_KEY_ID")
		if team == "" || kid == "" {
			return nil, errors.New("sign in with Apple needs AXON_APPLE_TEAM_ID and AXON_APPLE_KEY_ID")
		}
		out = append(out, Provider{ID: "apple", Name: "Apple", ClientID: id, //nolint:gosec // endpoints, not credentials
			AuthURL: "https://appleid.apple.com/auth/authorize", TokenURL: "https://appleid.apple.com/auth/token",
			Issuers: []string{"https://appleid.apple.com"}, appleTeam: team, appleKeyID: kid, appleKey: key})
	}
	if issuer := getenv("AXON_OIDC_ISSUER"); issuer != "" {
		id, secret, ok := pair("OIDC")
		if !ok {
			return nil, errors.New("AXON_OIDC_ISSUER needs AXON_OIDC_CLIENT_ID and AXON_OIDC_CLIENT_SECRET")
		}
		name := getenv("AXON_OIDC_NAME")
		if name == "" {
			name = "single sign-on"
		}
		p, err := discover(issuer)
		if err != nil {
			return nil, fmt.Errorf("AXON_OIDC_ISSUER: %w", err)
		}
		p.ID, p.Name, p.ClientID, p.Secret = "oidc", name, id, secret
		out = append(out, p)
	}
	return out, nil
}

// discover reads an OpenID provider's configuration once, at start.
func discover(issuer string) (Provider, error) {
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return Provider{}, errors.New("the issuer is an https address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration", nil)
	res, err := OAuthClient.Do(req)
	if err != nil {
		return Provider{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return Provider{}, fmt.Errorf("discovery answered %s", res.Status)
	}
	var doc struct {
		Issuer   string `json:"issuer"`
		Auth     string `json:"authorization_endpoint"`
		Token    string `json:"token_endpoint"`
		Userinfo string `json:"userinfo_endpoint"`
	}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return Provider{}, err
	}
	if doc.Issuer == "" || doc.Auth == "" || doc.Token == "" {
		return Provider{}, errors.New("the discovery document lacks endpoints")
	}
	return Provider{AuthURL: doc.Auth, TokenURL: doc.Token, Scopes: []string{"openid", "email", "profile"}, Issuers: []string{doc.Issuer}}, nil
}

func parseAppleKey(p string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(p))
	if block == nil {
		return nil, errors.New("AXON_APPLE_PRIVATE_KEY is the .p8 key Apple gave you, in PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("AXON_APPLE_PRIVATE_KEY: %w", err)
	}
	ec, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("AXON_APPLE_PRIVATE_KEY is not an EC key")
	}
	return ec, nil
}

// clientSecret is the secret sent to the token endpoint: Apple's is a JWT
// (ES256) valid a few minutes, signed with the operator's key.
func (p Provider) clientSecret(now time.Time) (string, error) {
	if p.appleKey == nil {
		return p.Secret, nil
	}
	enc := base64.RawURLEncoding
	head, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": p.appleKeyID})
	body, _ := json.Marshal(map[string]any{"iss": p.appleTeam, "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com", "sub": p.ClientID})
	signing := enc.EncodeToString(head) + "." + enc.EncodeToString(body)
	digest := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, p.appleKey, digest[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signing + "." + enc.EncodeToString(sig), nil
}

func (p Provider) issuerOK(iss string) bool {
	if p.issuerRe != nil {
		return p.issuerRe.MatchString(iss)
	}
	for _, i := range p.Issuers {
		if iss == i {
			return true
		}
	}
	return false
}
