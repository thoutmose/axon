// Package wsclient is a small WebSocket client (RFC 6455) for the chat
// services Axon listens to without being reachable from the internet
// (EP-108): Slack's Socket Mode and Discord's gateway. It reads text
// messages (reassembling fragments), answers pings, and writes masked text
// frames. Compression extensions are not offered.
package wsclient

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6455 defines the accept key with SHA-1
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MaxMessage bounds one message; chat events are far smaller.
const MaxMessage = 4 << 20

// Conn is an open WebSocket.
type Conn struct {
	conn net.Conn
	r    *bufio.Reader
	wmu  sync.Mutex
}

// ErrClosed is returned once the server closed the connection.
var ErrClosed = errors.New("websocket closed")

// TLSConfig is used for wss:// addresses; tests replace it.
var TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}

// Dial opens a WebSocket to a ws:// or wss:// address.
func Dial(ctx context.Context, rawURL string) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), map[string]string{"wss": "443", "ws": "80"}[u.Scheme])
	}
	var d net.Dialer
	var conn net.Conn
	switch u.Scheme {
	case "wss":
		cfg := TLSConfig.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = u.Hostname()
		}
		td := tls.Dialer{NetDialer: &d, Config: cfg}
		conn, err = td.DialContext(ctx, "tcp", host)
	case "ws":
		conn, err = d.DialContext(ctx, "tcp", host)
	default:
		return nil, fmt.Errorf("not a websocket address: %s", u.Scheme)
	}
	if err != nil {
		return nil, err
	}
	keyBytes := make([]byte, 16)
	_, _ = rand.Read(keyBytes)
	key := base64.StdEncoding.EncodeToString(keyBytes)
	path := u.RequestURI()
	req := "GET " + path + " HTTP/1.1\r\nHost: " + u.Host + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\nUser-Agent: Axon\r\nOrigin: https://" + u.Hostname() + "\r\n\r\n"
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	if _, err := io.WriteString(conn, req); err != nil {
		_ = conn.Close()
		return nil, err
	}
	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, &http.Request{Method: http.MethodGet})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = res.Body.Close()
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")) //nolint:gosec // see above
	if res.StatusCode != http.StatusSwitchingProtocols || !strings.EqualFold(res.Header.Get("Upgrade"), "websocket") ||
		res.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(h[:]) {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket handshake refused: %s", res.Status)
	}
	_ = conn.SetDeadline(time.Time{})
	return &Conn{conn: conn, r: r}, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	_ = c.writeFrame(0x8, []byte{0x03, 0xE8}) // 1000: normal closure
	return c.conn.Close()
}

// SetReadDeadline bounds the next Read.
func (c *Conn) SetReadDeadline(t time.Time) error { return c.conn.SetReadDeadline(t) }

// WriteText sends one text message.
func (c *Conn) WriteText(b []byte) error { return c.writeFrame(0x1, b) }

func (c *Conn) writeFrame(op byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	header := []byte{0x80 | op}
	n := len(payload)
	switch {
	case n < 126:
		header = append(header, 0x80|byte(n))
	case n <= 0xFFFF:
		header = append(header, 0x80|126, byte(n>>8), byte(n)) //nolint:gosec // n fits in 16 bits here
	default:
		header = append(header, 0x80|127)
		header = binary.BigEndian.AppendUint64(header, uint64(n))
	}
	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	header = append(header, mask...)
	masked := make([]byte, n)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	_, err := c.conn.Write(append(header, masked...))
	return err
}

// Read returns the next text or binary message; it answers pings itself.
func (c *Conn) Read() ([]byte, error) {
	var msg []byte
	for {
		head := make([]byte, 2)
		if _, err := io.ReadFull(c.r, head); err != nil {
			return nil, err
		}
		fin, op := head[0]&0x80 != 0, head[0]&0x0F
		if head[1]&0x80 != 0 {
			return nil, errors.New("websocket: the server masked a frame")
		}
		n := uint64(head[1] & 0x7F)
		switch n {
		case 126:
			ext := make([]byte, 2)
			if _, err := io.ReadFull(c.r, ext); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(ext))
		case 127:
			ext := make([]byte, 8)
			if _, err := io.ReadFull(c.r, ext); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(ext)
		}
		if n > MaxMessage || uint64(len(msg))+n > MaxMessage {
			return nil, errors.New("websocket: message too large")
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(c.r, payload); err != nil {
			return nil, err
		}
		switch op {
		case 0x8: // close
			return nil, ErrClosed
		case 0x9: // ping
			if err := c.writeFrame(0xA, payload); err != nil {
				return nil, err
			}
			continue
		case 0xA: // pong
			continue
		case 0x0, 0x1, 0x2: // continuation, text, binary
			msg = append(msg, payload...)
			if fin {
				return msg, nil
			}
		default:
			return nil, fmt.Errorf("websocket: unknown opcode %d", op)
		}
	}
}
