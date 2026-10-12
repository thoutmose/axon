package textx

import (
	"testing"
	"unicode/utf8"
)

func TestCut(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 2, "he"},
		{"héllo", 2, "h"},     // é is two bytes: never split it
		{"héllo", 3, "hé"},    // fits exactly
		{"日本語", 4, "日"},       // three-byte characters
		{"a\xffb", 10, "a�b"}, // invalid bytes become U+FFFD
		{"", 3, ""},
		{"abc", 0, ""},
	} {
		got := Cut(c.in, c.n)
		if got != c.want || len(got) > max(c.n, len(c.want)) || !utf8.ValidString(got) {
			t.Errorf("Cut(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestCutNeverExceedsTheLimit(t *testing.T) {
	// Replacing an invalid byte (1 byte) with U+FFFD (3 bytes) must not
	// push the result over n.
	got := Cut("\xff\xff\xff\xff", 4)
	if len(got) > 4 || !utf8.ValidString(got) {
		t.Errorf("Cut = %q (%d bytes), want valid UTF-8 within 4 bytes", got, len(got))
	}
}
