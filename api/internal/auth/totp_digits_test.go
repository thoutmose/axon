package auth

import "testing"

// An authenticator code is read as its digits, however it was typed.
func TestTOTPDigits(t *testing.T) {
	for in, want := range map[string]string{"123456": "123456", "123 456": "123456", " 123456 ": "123456", "123-456": "123456", "12a456": "12456"} {
		if got := totpDigits(in); got != want {
			t.Errorf("totpDigits(%q) = %q, want %q", in, got, want)
		}
	}
}
