package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// softAuthenticator is a minimal WebAuthn platform authenticator for tests:
// one P-256 discoverable credential, "none" attestation.
type softAuthenticator struct {
	t          *testing.T
	origin     string
	key        *ecdsa.PrivateKey
	credID     []byte
	userHandle []byte
	signCount  uint32
}

var b64 = base64.RawURLEncoding

const credIDLen = 16

type options struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
		RPID      string `json:"rpId"`
		RP        struct {
			ID string `json:"id"`
		} `json:"rp"`
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"publicKey"`
}

func (a *softAuthenticator) clientData(typ, challenge string) []byte {
	b, _ := json.Marshal(map[string]any{"type": typ, "challenge": challenge, "origin": a.origin, "crossOrigin": false})
	return b
}

// create answers navigator.credentials.create() with a new credential.
func (a *softAuthenticator) create(optionsJSON []byte) []byte {
	a.t.Helper()
	var o options
	if err := json.Unmarshal(optionsJSON, &o); err != nil {
		a.t.Fatalf("creation options: %v (%s)", err, optionsJSON)
	}
	var err error
	if a.key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
		a.t.Fatal(err)
	}
	a.credID = make([]byte, credIDLen)
	_, _ = rand.Read(a.credID)
	if a.userHandle, err = b64.DecodeString(o.PublicKey.User.ID); err != nil {
		a.t.Fatal(err)
	}

	pub, err := a.key.PublicKey.ECDH()
	if err != nil {
		a.t.Fatal(err)
	}
	raw := pub.Bytes() // 0x04 || X || Y
	cose, _ := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: raw[1:33], -3: raw[33:65]})

	rpHash := sha256.Sum256([]byte(o.PublicKey.RP.ID))
	auth := append([]byte{}, rpHash[:]...)
	auth = append(auth, 0x45)                // UP | UV | AT
	auth = append(auth, 0, 0, 0, 0)          // sign count
	auth = append(auth, make([]byte, 16)...) // AAGUID
	auth = binary.BigEndian.AppendUint16(auth, credIDLen)
	auth = append(auth, a.credID...)
	auth = append(auth, cose...)
	attObj, _ := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": auth})

	out, _ := json.Marshal(map[string]any{
		"id": b64.EncodeToString(a.credID), "rawId": b64.EncodeToString(a.credID), "type": "public-key",
		"authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(a.clientData("webauthn.create", o.PublicKey.Challenge)),
			"attestationObject": b64.EncodeToString(attObj),
			"transports":        []string{"internal"},
		},
	})
	return out
}

// get answers navigator.credentials.get() by signing the challenge.
func (a *softAuthenticator) get(optionsJSON []byte) []byte {
	a.t.Helper()
	var o options
	if err := json.Unmarshal(optionsJSON, &o); err != nil {
		a.t.Fatalf("request options: %v (%s)", err, optionsJSON)
	}
	a.signCount++
	rpHash := sha256.Sum256([]byte(o.PublicKey.RPID))
	auth := append([]byte{}, rpHash[:]...)
	auth = append(auth, 0x05) // UP | UV
	auth = binary.BigEndian.AppendUint32(auth, a.signCount)

	cd := a.clientData("webauthn.get", o.PublicKey.Challenge)
	cdHash := sha256.Sum256(cd)
	digest := sha256.Sum256(append(append([]byte{}, auth...), cdHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		a.t.Fatal(err)
	}
	out, _ := json.Marshal(map[string]any{
		"id": b64.EncodeToString(a.credID), "rawId": b64.EncodeToString(a.credID), "type": "public-key",
		"authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(cd),
			"authenticatorData": b64.EncodeToString(auth),
			"signature":         b64.EncodeToString(sig),
			"userHandle":        b64.EncodeToString(a.userHandle),
		},
	})
	return out
}
