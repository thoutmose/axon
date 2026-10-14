package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type certJSON struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	ExamOn    *string `json:"exam_on"`
	PassedOn  *string `json:"passed_on"`
	ExpiresOn *string `json:"expires_on"`
	TargetOn  *string `json:"target_on"`
	CostCents *int    `json:"cost_cents"`
}

// Certifications (EP-104): planned, booked, passed, or failed; passing
// sets the expiry from the validity; exams and expiries on the calendar;
// never someone else's.
func TestCertifications(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	in10 := time.Now().UTC().AddDate(0, 0, 10).Format(time.DateOnly)

	var c certJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/certifications", []byte(`{"name":"Certified Kubernetes Administrator","provider":"The Linux Foundation",
		"code":"CKA","url":"https://training.linuxfoundation.org/certification/certified-kubernetes-administrator-cka/","status":"booked","exam_on":"`+in10+`",
		"cost_cents":44500,"currency":"USD","validity_months":24}`)), &c)
	if c.Status != "booked" || c.ExamOn == nil || *c.ExamOn != in10 {
		t.Fatalf("created = %+v", c)
	}
	for _, bad := range []string{
		`{"name":"","status":"planned"}`,
		`{"name":"X","status":"dreaming"}`,
		`{"name":"X","status":"booked"}`,
		`{"name":"X","status":"passed"}`,
		`{"name":"X","status":"planned","cost_cents":-1}`,
		`{"name":"X","status":"planned","url":"ftp://x"}`,
		`{"name":"X","status":"planned","validity_months":0}`,
	} {
		alice.mustDo(422, "POST", "/v1/certifications", []byte(bad))
	}

	// Passing sets the expiry from the validity.
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/certifications/"+c.ID, []byte(`{"name":"Certified Kubernetes Administrator","provider":"The Linux Foundation",
		"code":"CKA","status":"passed","exam_on":"2026-09-01","passed_on":"2026-09-01","score":"89 %","cost_cents":44500,"currency":"USD","validity_months":24}`)), &c)
	if c.Status != "passed" || c.ExpiresOn == nil || *c.ExpiresOn != "2028-09-01" {
		t.Fatalf("passed = %+v", c)
	}
	var cal struct {
		Items []struct {
			Date  string `json:"date"`
			Type  string `json:"type"`
			Kind  string `json:"kind"`
			Title string `json:"title"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/calendar?types=certification&from=2026-09-01&to=2026-09-30", nil), &cal)
	if len(cal.Items) != 1 || cal.Items[0].Kind != "exam" || !strings.Contains(cal.Items[0].Title, "CKA") {
		t.Fatalf("calendar = %+v", cal.Items)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/calendar?types=certification&from=2028-08-15&to=2028-09-15", nil), &cal)
	if len(cal.Items) != 1 || cal.Items[0].Kind != "expiry" {
		t.Fatalf("expiry on the calendar = %+v", cal.Items)
	}

	bob.mustDo(404, "PUT", "/v1/certifications/"+c.ID, []byte(`{"name":"x","status":"planned"}`))
	bob.mustDo(404, "DELETE", "/v1/certifications/"+c.ID, nil)
	var list struct {
		Items []certJSON `json:"items"`
	}
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/certifications", nil), &list)
	if len(list.Items) != 0 {
		t.Fatal("bob sees alice's certifications")
	}
	alice.mustDo(204, "DELETE", "/v1/certifications/"+c.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/certifications/"+c.ID, nil)
}
