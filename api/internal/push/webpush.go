// Package push sends Web Push notifications (EP-56) with the standard
// library only: payloads are encrypted for the browser (RFC 8291,
// aes128gcm), and requests are signed with the instance's VAPID key
// (RFC 8292). The push service in between (the browser vendor's) sees
// neither the content nor who the person is.
package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Subscription is what a browser's PushManager returns.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"` // the browser's public key, base64url
	Auth     string `json:"auth"`   // the browser's auth secret, base64url
}

// ErrGone means the subscription no longer exists (404 or 410): forget it.
var ErrGone = errors.New("push subscription gone")

var b64 = base64.RawURLEncoding

func decode(s string) ([]byte, error) {
	if b, err := b64.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s) // some browsers pad
}

// encrypt seals payload for one browser (RFC 8291 §3, one aes128gcm record).
// The ephemeral key and salt are parameters so the RFC's example can be checked.
func encrypt(uaPublic, authSecret, payload []byte, as *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	ua, err := ecdh.P256().NewPublicKey(uaPublic)
	if err != nil {
		return nil, fmt.Errorf("browser key: %w", err)
	}
	shared, err := as.ECDH(ua)
	if err != nil {
		return nil, err
	}
	asPublic := as.PublicKey().Bytes()
	prkKey, err := hkdf.Extract(sha256.New, shared, authSecret)
	if err != nil {
		return nil, err
	}
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPublic...), asPublic...)
	ikm, err := hkdf.Expand(sha256.New, prkKey, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, append(append([]byte{}, payload...), 2), nil) // 2: the last record
	var out bytes.Buffer
	out.Write(salt)
	_ = binary.Write(&out, binary.BigEndian, uint32(4096))
	out.WriteByte(65) // an uncompressed P-256 public key
	out.Write(asPublic)
	out.Write(sealed)
	return out.Bytes(), nil
}

// Encrypt seals payload for the subscription with a fresh key and salt.
func Encrypt(sub Subscription, payload []byte) ([]byte, error) {
	ua, err := decode(sub.P256dh)
	if err != nil {
		return nil, fmt.Errorf("browser key: %w", err)
	}
	auth, err := decode(sub.Auth)
	if err != nil || len(auth) != 16 {
		return nil, errors.New("browser auth secret: not 16 bytes")
	}
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return encrypt(ua, auth, payload, as, salt)
}

// VAPID is the instance's signing key for push requests.
type VAPID struct {
	Key     *ecdsa.PrivateKey
	Subject string // mailto: or https: contact for push services
}

// PublicKey is the key browsers subscribe with (applicationServerKey).
func (v VAPID) PublicKey() string {
	pub, err := v.Key.PublicKey.ECDH()
	if err != nil {
		return ""
	}
	return b64.EncodeToString(pub.Bytes())
}

// token signs the JWT a push service checks (RFC 8292 §2, ES256).
func (v VAPID) token(endpoint string, now time.Time) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	header := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, _ := json.Marshal(map[string]any{"aud": u.Scheme + "://" + u.Host, "exp": now.Add(12 * time.Hour).Unix(), "sub": v.Subject})
	signing := header + "." + b64.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, v.Key, digest[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signing + "." + b64.EncodeToString(sig), nil
}

// NewKey makes a VAPID key.
func NewKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// Send delivers one encrypted notification. It returns ErrGone when the
// browser unsubscribed, so the caller can forget the subscription.
func Send(ctx context.Context, hc *http.Client, v VAPID, sub Subscription, payload []byte, ttl time.Duration) error {
	body, err := Encrypt(sub, payload)
	if err != nil {
		return err
	}
	jwt, err := v.token(sub.Endpoint, time.Now())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", fmt.Sprint(int(ttl.Seconds())))
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+v.PublicKey())
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return ErrGone
	case res.StatusCode >= 300:
		return fmt.Errorf("push service answered %d", res.StatusCode)
	}
	return nil
}
