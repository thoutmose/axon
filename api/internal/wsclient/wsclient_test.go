package wsclient

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// The client talks to a standard server: text both ways, large messages,
// a close.
func TestAgainstAServer(t *testing.T) {
	srv := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		_ = websocket.Message.Send(ws, `{"hello":1}`)
		for {
			var in string
			if err := websocket.Message.Receive(ws, &in); err != nil {
				return
			}
			if in == "big" {
				_ = websocket.Message.Send(ws, strings.Repeat("x", 70000))
				continue
			}
			if in == "bye" {
				_ = ws.Close()
				return
			}
			_ = websocket.Message.Send(ws, "echo:"+in)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/socket?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(); err != nil || string(m) != `{"hello":1}` {
		t.Fatalf("hello = %q %v", m, err)
	}
	for _, s := range []string{"a", strings.Repeat("é", 200), strings.Repeat("y", 70000)} {
		if err := c.WriteText([]byte(s)); err != nil {
			t.Fatal(err)
		}
		if m, err := c.Read(); err != nil || string(m) != "echo:"+s {
			t.Fatalf("echo of %d bytes = %d bytes, %v", len(s), len(m), err)
		}
	}
	_ = c.WriteText([]byte("big"))
	if m, err := c.Read(); err != nil || len(m) != 70000 {
		t.Fatalf("big = %d %v", len(m), err)
	}
	_ = c.WriteText([]byte("bye"))
	if _, err := c.Read(); err == nil {
		t.Fatal("read after close")
	}
	_ = c.Close()
	if _, err := Dial(ctx, srv.URL); err == nil {
		t.Fatal("dialled an http:// address")
	}
}
