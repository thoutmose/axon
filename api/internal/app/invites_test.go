package app

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// Invitations (EP-120): the owner of an Axon whose registration is closed
// invites one person with a link, used once, that can bring them into the
// owner's household; nobody else can invite, and a used, revoked, or
// expired link opens nothing.
func TestInvitations(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(201, "POST", "/v1/household", []byte(`{"name":"Home"}`))

	stranger := e.client()
	stranger.mustDo(403, "POST", "/v1/auth/register/begin", []byte(`{"display_name":"Mallory"}`))

	type inviteJSON struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Note      string `json:"note"`
		Household bool   `json:"household"`
	}
	var inv inviteJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/invites", []byte(`{"note":"For Sam","household":true,"days":7}`)), &inv)
	u, err := url.Parse(inv.URL)
	if err != nil || u.Path != "/register" || len(u.Query().Get("invite")) < 40 || !inv.Household {
		t.Fatalf("invite = %+v", inv)
	}
	token := u.Query().Get("invite")
	for _, bad := range []string{`{"days":0}`, `{"days":31}`, `{"note":"` + strings.Repeat("x", 201) + `"}`} {
		alice.mustDo(422, "POST", "/v1/invites", []byte(bad))
	}

	// Anyone with the link sees who invites them; a wrong link, nothing.
	var who struct {
		Inviter   string `json:"inviter"`
		Household bool   `json:"household"`
	}
	_ = json.Unmarshal(stranger.mustDo(200, "GET", "/v1/auth/invite?token="+token, nil), &who)
	if who.Inviter != "Alice" || !who.Household {
		t.Fatalf("invite seen = %+v", who)
	}
	stranger.mustDo(404, "GET", "/v1/auth/invite?token=nope", nil)

	// Sam signs up with it, and is in Alice's household.
	sam := e.client()
	opts := sam.mustDo(200, "POST", "/v1/auth/register/begin", []byte(`{"display_name":"Sam","invite":"`+token+`"}`))
	sam.mustDo(201, "POST", "/v1/auth/register/finish", sam.key.create(opts))
	var home struct {
		Members []struct {
			DisplayName string `json:"display_name"`
		} `json:"members"`
	}
	_ = json.Unmarshal(sam.mustDo(200, "GET", "/v1/household", nil), &home)
	if len(home.Members) != 2 {
		t.Fatalf("household = %+v", home)
	}

	// The link worked once; Sam cannot invite.
	stranger.mustDo(404, "GET", "/v1/auth/invite?token="+token, nil)
	stranger.mustDo(403, "POST", "/v1/auth/register/begin", []byte(`{"display_name":"Mallory","invite":"`+token+`"}`))
	sam.mustDo(403, "POST", "/v1/invites", []byte(`{}`))
	var list struct {
		Items []struct {
			ID     string  `json:"id"`
			UsedBy *string `json:"used_by_name"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/invites", nil), &list)
	if len(list.Items) != 1 || list.Items[0].UsedBy == nil || *list.Items[0].UsedBy != "Sam" {
		t.Fatalf("invites = %+v", list.Items)
	}

	// A revoked link opens nothing.
	var second inviteJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/invites", []byte(`{}`)), &second)
	alice.mustDo(204, "DELETE", "/v1/invites/"+second.ID, nil)
	u2, _ := url.Parse(second.URL)
	stranger.mustDo(404, "GET", "/v1/auth/invite?token="+u2.Query().Get("invite"), nil)
	alice.mustDo(404, "DELETE", "/v1/invites/"+inv.ID, nil) // used: kept as a record, not revocable
}
