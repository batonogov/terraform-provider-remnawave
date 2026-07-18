package provider

import "testing"

// TestNormalizeUserAction verifies that alias spellings are rewritten to
// their canonical (underscore) form, and canonical inputs pass through
// unchanged.
func TestNormalizeUserAction(t *testing.T) {
	cases := []struct {
		in          string
		want        string
		wantWarning bool
	}{
		{"enable", "enable", false},
		{"disable", "disable", false},
		{"reset_traffic", "reset_traffic", false},
		{"reset-traffic", "reset_traffic", true}, // deprecated alias
		{"revoke_subscription", "revoke_subscription", false},
		{"enable_unknown", "enable_unknown", false}, // unknown — returned as-is, no warning
	}
	for _, c := range cases {
		got, warned := normalizeUserAction(c.in)
		if got != c.want {
			t.Errorf("normalizeUserAction(%q): got %q, want %q", c.in, got, c.want)
		}
		if warned != c.wantWarning {
			t.Errorf("normalizeUserAction(%q): warning flag got %v, want %v", c.in, warned, c.wantWarning)
		}
	}
}

// TestIsValidUserActionAcceptsAlias confirms both the canonical and the
// deprecated alias spellings are accepted by the validator.
func TestIsValidUserActionAcceptsAlias(t *testing.T) {
	for _, a := range []string{"enable", "disable", "reset_traffic", "reset-traffic", "revoke_subscription"} {
		if !isValidUserAction(a) {
			t.Errorf("isValidUserAction(%q) = false, want true", a)
		}
	}
	for _, a := range []string{"", "bogus", "RESET_TRAFFIC", "reset traffic", "revoke", "restart"} {
		if isValidUserAction(a) {
			t.Errorf("isValidUserAction(%q) = true, want false", a)
		}
	}
}

// TestNormalizeNodeAction verifies the node-action alias path.
func TestNormalizeNodeAction(t *testing.T) {
	cases := []struct {
		in          string
		want        string
		wantWarning bool
	}{
		{"enable", "enable", false},
		{"disable", "disable", false},
		{"restart", "restart", false},
		{"reset_traffic", "reset_traffic", false},
		{"reset-traffic", "reset_traffic", true}, // deprecated alias
		{"reset_traffic_unknown", "reset_traffic_unknown", false},
	}
	for _, c := range cases {
		got, warned := normalizeNodeAction(c.in)
		if got != c.want {
			t.Errorf("normalizeNodeAction(%q): got %q, want %q", c.in, got, c.want)
		}
		if warned != c.wantWarning {
			t.Errorf("normalizeNodeAction(%q): warning flag got %v, want %v", c.in, warned, c.wantWarning)
		}
	}
}
