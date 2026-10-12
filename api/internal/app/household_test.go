package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type householdJSON struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members []struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Owner  bool   `json:"owner"`
	} `json:"members"`
	Lists []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Open  int    `json:"open"`
		Items []struct {
			ID       string  `json:"id"`
			Title    string  `json:"title"`
			Assignee *string `json:"assignee"`
			DoneBy   *string `json:"done_by"`
		} `json:"items"`
	} `json:"lists"`
}

func (c *client) household(want int, method, path string, body string) householdJSON {
	c.e.t.Helper()
	var h householdJSON
	_ = json.Unmarshal(c.mustDo(want, method, path, []byte(body)), &h)
	return h
}

// A household shares lists among people who joined with a code, and
// nothing else; anyone can leave, and the owner can remove members.
func TestHousehold(t *testing.T) {
	e := newEnv(t, true)
	alice, bob, carol := e.client(), e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	carol.register("Carol")

	alice.mustDo(404, "GET", "/v1/household", nil)
	h := alice.household(201, "POST", "/v1/household", `{"name":"Rue des Lilas"}`)
	if h.Name != "Rue des Lilas" || len(h.Members) != 1 || !h.Members[0].Owner {
		t.Fatalf("household = %+v", h)
	}
	alice.mustDo(409, "POST", "/v1/household", []byte(`{"name":"Second"}`))

	// A one-time code brings Bob in; used, it does not work again.
	var inv struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/household/invites", nil), &inv)
	bob.mustDo(422, "POST", "/v1/household/join", []byte(`{"code":"WRONG-CODE-0000-0000"}`))
	h = bob.household(200, "POST", "/v1/household/join", fmt.Sprintf(`{"code":%q}`, strings.ToLower(inv.Code)))
	if len(h.Members) != 2 || h.Members[1].Name != "Bob" {
		t.Fatalf("after joining = %+v", h)
	}
	carol.mustDo(422, "POST", "/v1/household/join", []byte(fmt.Sprintf(`{"code":%q}`, inv.Code)))
	carol.mustDo(404, "GET", "/v1/household", nil)

	// Shared lists: Bob adds, Alice sees and ticks; items can be given to a member, not to an outsider.
	h = bob.household(201, "POST", "/v1/household/lists", `{"title":"Groceries","kind":"shopping"}`)
	list := h.Lists[0].ID
	h = bob.household(201, "POST", "/v1/household/lists/"+list+"/items", `{"title":"Oat milk","assignee":"`+h.Members[0].UserID+`"}`)
	bob.mustDo(422, "POST", "/v1/household/lists/"+list+"/items", []byte(`{"title":"Eggs","assignee":"`+carolID(t, carol)+`"}`))
	carol.mustDo(404, "POST", "/v1/household/lists/"+list+"/items", []byte(`{"title":"Intruder"}`))
	h = alice.household(200, "GET", "/v1/household", "")
	if len(h.Lists) != 1 || h.Lists[0].Open != 1 || h.Lists[0].Items[0].Title != "Oat milk" || h.Lists[0].Items[0].Assignee == nil {
		t.Fatalf("alice's view = %+v", h.Lists)
	}
	item := h.Lists[0].Items[0].ID
	carol.mustDo(404, "PATCH", "/v1/household/items/"+item, []byte(`{"done":true}`))
	h = alice.household(200, "PATCH", "/v1/household/items/"+item, `{"done":true}`)
	if h.Lists[0].Open != 0 || h.Lists[0].Items[0].DoneBy == nil {
		t.Fatalf("after ticking = %+v", h.Lists[0])
	}

	// Rename, delete an item, add and remove a list.
	if h = bob.household(200, "PATCH", "/v1/household", `{"name":"The Lilacs"}`); h.Name != "The Lilacs" {
		t.Fatalf("renamed = %+v", h)
	}
	h = bob.household(201, "POST", "/v1/household/lists/"+list+"/items", `{"title":"Bread"}`)
	bread := ""
	for _, it := range h.Lists[0].Items {
		if it.Title == "Bread" {
			bread = it.ID
		}
	}
	bob.mustDo(204, "DELETE", "/v1/household/items/"+bread, nil)
	h = bob.household(201, "POST", "/v1/household/lists", `{"title":"Chores","kind":"chores"}`)
	chores := h.Lists[1].ID
	if h = bob.household(200, "PATCH", "/v1/household/lists/"+chores, `{"title":"Weekly chores"}`); h.Lists[1].Title != "Weekly chores" {
		t.Fatalf("list renamed = %+v", h.Lists)
	}
	carol.mustDo(404, "DELETE", "/v1/household/lists/"+chores, nil)
	bob.mustDo(204, "DELETE", "/v1/household/lists/"+chores, nil)

	// Nothing personal is shared.
	alice.createTask(`{"title":"Alice's private task"}`)
	if out := bob.mustDo(200, "GET", "/v1/tasks?view=all", nil); bytes.Contains(out, []byte("private task")) {
		t.Fatal("a personal task leaked to the household")
	}

	// Only the owner removes members; an owner who leaves hands the household on.
	bob.mustDo(403, "DELETE", "/v1/household/members/"+h.Members[0].UserID, nil)
	alice.mustDo(204, "DELETE", "/v1/household/me", nil)
	h = bob.household(200, "GET", "/v1/household", "")
	if len(h.Members) != 1 || !h.Members[0].Owner || h.Lists[0].Items[0].Assignee != nil {
		t.Fatalf("after alice left = %+v", h)
	}
	bob.mustDo(204, "DELETE", "/v1/household/me", nil) // the last one out
	bob.mustDo(404, "GET", "/v1/household", nil)
}

func carolID(t *testing.T, c *client) string {
	t.Helper()
	var me struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me", nil), &me)
	return me.ID
}
