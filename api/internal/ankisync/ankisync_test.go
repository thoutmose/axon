package ankisync

import "testing"

func TestSummaryKeepsTheFirstLine(t *testing.T) {
	in := "A network error occurred.\n\nError details: \u2068HTTP status client error (405 Method Not Allowed) for url ()\u2069"
	if got := summary(in); got != "A network error occurred." {
		t.Fatalf("summary = %q", got)
	}
}
