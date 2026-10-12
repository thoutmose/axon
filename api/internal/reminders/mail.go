package reminders

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// Mailer sends reminders by email through the operator's mail server:
// AXON_SMTP_URL as smtp://user:pass@host:587 (STARTTLS when offered) or
// smtps://user:pass@host:465 (TLS from the start), from AXON_SMTP_FROM.
type Mailer struct {
	host, port, user, pass, from, base string
	implicitTLS                        bool
}

// NewMailer returns nil when no mail server is configured.
func NewMailer(rawURL, from, base string) (*Mailer, error) {
	if rawURL == "" {
		return nil, nil //nolint:nilnil // no mail server: email reminders are off
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "smtp" && u.Scheme != "smtps") || u.Hostname() == "" {
		return nil, fmt.Errorf("AXON_SMTP_URL must look like smtp://user:pass@host:587 or smtps://user:pass@host:465")
	}
	if from == "" || strings.ContainsAny(from, "\r\n") {
		return nil, fmt.Errorf("AXON_SMTP_FROM must be set to the sender address")
	}
	m := &Mailer{host: u.Hostname(), port: u.Port(), from: from, base: strings.TrimRight(base, "/"), implicitTLS: u.Scheme == "smtps"}
	if m.port == "" {
		m.port = map[bool]string{true: "465", false: "587"}[m.implicitTLS]
	}
	if u.User != nil {
		m.user = u.User.Username()
		m.pass, _ = u.User.Password()
	}
	return m, nil
}

// compose writes the message: plain text, with a link back to Axon.
func (m *Mailer) compose(to string, msg Message, now time.Time) []byte {
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domain := m.from[strings.LastIndex(m.from, "@")+1:]
	var b strings.Builder
	fmt.Fprintf(&b, "From: Axon <%s>\r\nTo: <%s>\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@%s>\r\n",
		m.from, to, mime.QEncoding.Encode("utf-8", msg.Title), now.Format(time.RFC1123Z), hex.EncodeToString(id), domain)
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\nAuto-Submitted: auto-generated\r\n\r\n")
	b.WriteString(strings.ReplaceAll(msg.Body, "\n", "\r\n"))
	if m.base != "" {
		fmt.Fprintf(&b, "\r\n\r\n%s%s\r\n", m.base, msg.URL)
	}
	b.WriteString("\r\n-- \r\nTurn reminders off in Axon: Account → Preferences → Reminders.\r\n")
	return []byte(b.String())
}

// Send delivers one reminder.
func (m *Mailer) Send(to string, msg Message) error {
	if strings.ContainsAny(to, "\r\n<>") {
		return fmt.Errorf("bad address")
	}
	addr := net.JoinHostPort(m.host, m.port)
	var c *smtp.Client
	var err error
	if m.implicitTLS {
		conn, derr := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12})
		if derr != nil {
			return derr
		}
		c, err = smtp.NewClient(conn, m.host)
	} else {
		c, err = smtp.Dial(addr)
		if err == nil {
			if ok, _ := c.Extension("STARTTLS"); ok {
				err = c.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12})
			}
		}
	}
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if m.user != "" {
		if err := c.Auth(smtp.PlainAuth("", m.user, m.pass, m.host)); err != nil {
			return err
		}
	}
	if err := c.Mail(m.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(m.compose(to, msg, time.Now())); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}
