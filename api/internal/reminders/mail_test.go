package reminders

import (
	"strings"
	"testing"
	"time"
)

func TestMailer(t *testing.T) {
	if m, err := NewMailer("", "", ""); m != nil || err != nil {
		t.Fatal("no mail server should mean no mailer")
	}
	for _, bad := range []string{"http://mail.example", "smtp://", "smtp://mail.example"} {
		if _, err := NewMailer(bad, map[bool]string{true: "", false: "axon@example.org"}[bad == "smtp://mail.example"], ""); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	m, err := NewMailer("smtps://axon:secret@mail.example", "axon@example.org", "https://axon.example/")
	if err != nil || m.port != "465" || !m.implicitTLS || m.user != "axon" || m.pass != "secret" {
		t.Fatalf("mailer = %+v, %v", m, err)
	}
	msg := m.compose("ada@example.org", Message{Title: "Today in Axon", Body: "1 task · 3 cards", URL: "/today"}, time.Unix(0, 0))
	s := string(msg)
	for _, want := range []string{"From: Axon <axon@example.org>\r\n", "To: <ada@example.org>\r\n", "Subject: Today in Axon\r\n",
		"Content-Type: text/plain; charset=utf-8", "Auto-Submitted: auto-generated", "1 task · 3 cards", "https://axon.example/today"} {
		if !strings.Contains(s, want) {
			t.Errorf("message lacks %q:\n%s", want, s)
		}
	}
	if err := m.Send("ada@example.org\r\nBcc: eve@example.org", Message{}); err == nil {
		t.Error("an address with a line break was accepted")
	}
}
