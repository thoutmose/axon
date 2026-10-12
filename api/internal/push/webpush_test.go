package push

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

type bigInt = big.Int

func mustB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := b64.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The example in RFC 8291 §5, byte for byte.
func TestEncryptMatchesRFC8291(t *testing.T) {
	as, err := ecdh.P256().NewPrivateKey(mustB64(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	if got := b64.EncodeToString(as.PublicKey().Bytes()); got != "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8" {
		t.Fatalf("sender public key = %s", got)
	}
	out, err := encrypt(
		mustB64(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"),
		mustB64(t, "BTBZMqHH6r4Tts7J_aSIgg"),
		[]byte("When I grow up, I want to be a watermelon"),
		as, mustB64(t, "DGv6ra1nlYgDCS1FRnbzlw"))
	if err != nil {
		t.Fatal(err)
	}
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if got := b64.EncodeToString(out); got != want {
		t.Fatalf("encrypted =\n%s\nwant\n%s", got, want)
	}
}

func TestVAPIDToken(t *testing.T) {
	key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	v := VAPID{Key: key, Subject: "https://axon.example"}
	tok, err := v.token("https://push.example.net/send/abc", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token = %s", tok)
	}
	var claims map[string]any
	_ = json.Unmarshal(mustB64(t, parts[1]), &claims)
	if claims["aud"] != "https://push.example.net" || claims["sub"] != "https://axon.example" {
		t.Fatalf("claims = %v", claims)
	}
	sig := mustB64(t, parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r, s := new(bigInt).SetBytes(sig[:32]), new(bigInt).SetBytes(sig[32:])
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
		t.Fatal("signature does not verify")
	}
	if len(mustB64(t, v.PublicKey())) != 65 {
		t.Fatal("public key is not an uncompressed P-256 point")
	}
}
