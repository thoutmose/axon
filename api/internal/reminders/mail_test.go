package reminders

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
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

// A digest email parses as a mail app reads it: text, then HTML with the
// picture it shows inline.
func TestRichMail(t *testing.T) {
	m, _ := NewMailer("smtp://mail.example", "axon@example.org", "")
	raw := m.composeRich("ada@example.org", Rich{Subject: "Votre journée · mercredi", Text: "1. Écrire", HTML: "<p>1. Écrire</p>", PNG: []byte("\x89PNG")}, time.Unix(0, 0))
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if subj, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject")); subj != "Votre journée · mercredi" {
		t.Fatalf("subject = %q", subj)
	}
	_, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	parts := multipart.NewReader(msg.Body, params["boundary"])
	text, _ := parts.NextPart()
	body, _ := io.ReadAll(base64.NewDecoder(base64.StdEncoding, text))
	if text.Header.Get("Content-Type") != "text/plain; charset=utf-8" || string(body) != "1. Écrire" {
		t.Fatalf("text part %q %q", text.Header, body)
	}
	rel, _ := parts.NextPart()
	_, rp, _ := mime.ParseMediaType(rel.Header.Get("Content-Type"))
	inner := multipart.NewReader(rel, rp["boundary"])
	html, _ := inner.NextPart()
	img, _ := inner.NextPart()
	if !strings.HasPrefix(html.Header.Get("Content-Type"), "text/html") || img.Header.Get("Content-ID") != "<scene>" {
		t.Fatalf("related parts %v %v", html.Header, img.Header)
	}
	if _, err := parts.NextPart(); err != io.EOF {
		t.Fatalf("more parts: %v", err)
	}
}
