package main

import (
	"strings"
	"testing"
)

func TestUsageIsQRFirstAndDoesNotAdvertisePasswordAuth(t *testing.T) {
	text := usageText()
	for _, want := range []string{"--qr", "--access-token", "STEAM_USERNAME"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage missing %q in:\n%s", want, text)
		}
	}
	for _, removed := range []string{"--password", "--two-factor-code", "--auth-code", "STEAM_PASSWORD", "STEAM_TFA_CODE", "STEAM_AUTH_CODE"} {
		if strings.Contains(text, removed) {
			t.Fatalf("usage should not advertise password/TFA auth %q in:\n%s", removed, text)
		}
	}
}
