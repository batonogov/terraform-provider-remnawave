package provider

import (
	"testing"
)

// TestAccPasskeyResource_ImportSkip is a placeholder that documents the
// passkey import path. WebAuthn registration is interactive, so this test
// is always skipped in automated CI.
func TestAccPasskeyResource_ImportSkip(t *testing.T) {
	testAccPreCheck(t)
	t.Skip("requires pre-existing passkey UUID via env var; skipping automated run")
}
