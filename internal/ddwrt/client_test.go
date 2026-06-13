// SPDX-License-Identifier: AGPL-3.0-or-later

package ddwrt

import "testing"

func TestSplitHostPort(t *testing.T) {
	cases := []struct {
		in         string
		host, port string
	}{
		{"192.168.1.1", "192.168.1.1", ""},
		{"192.168.1.1:2222", "192.168.1.1", "2222"},
		{"ssh://router:22", "router", "22"},
		{" router ", "router", ""},
		{"router:notaport", "router:notaport", ""},
	}
	for _, tc := range cases {
		h, p := splitHostPort(tc.in)
		if h != tc.host || p != tc.port {
			t.Errorf("splitHostPort(%q) = (%q,%q), want (%q,%q)", tc.in, h, p, tc.host, tc.port)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain":          "'plain'",
		"with space":     "'with space'",
		"it's":           `'it'\''s'`,
		"a=b;rm -rf /":   "'a=b;rm -rf /'",
		"$(whoami)":      "'$(whoami)'",
		"back`tick`":     "'back`tick`'",
		`"double"`:       `'"double"'`,
		"192.168.1.1/24": "'192.168.1.1/24'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFullRestartService(t *testing.T) {
	full := []string{"*", "all", "rc", " rc ", "ALL"}
	for _, s := range full {
		// "ALL" is intentionally NOT a sentinel — match must be exact (lower-case).
		want := s != "ALL"
		if got := fullRestartService(s); got != want {
			t.Errorf("fullRestartService(%q) = %v, want %v", s, got, want)
		}
	}
	named := []string{"wan", "nas", "dnsmasq", "firewall", "httpd"}
	for _, s := range named {
		if fullRestartService(s) {
			t.Errorf("fullRestartService(%q) = true, want false (named service)", s)
		}
	}
}

func TestRestartCommand(t *testing.T) {
	c := NewClient(Config{Host: "10.0.0.1"})

	// Empty service -> no-op (empty command).
	if got := c.restartCommand(""); got != "" {
		t.Errorf("restartCommand(\"\") = %q, want \"\" (no-op)", got)
	}
	if got := c.restartCommand("   "); got != "" {
		t.Errorf("restartCommand(spaces) = %q, want \"\" (no-op)", got)
	}

	// Full-restart sentinels -> rc restart.
	for _, s := range []string{"*", "all", "rc"} {
		if got := c.restartCommand(s); got != fullRestart {
			t.Errorf("restartCommand(%q) = %q, want %q", s, got, fullRestart)
		}
	}

	// Named service -> default stopservice/startservice template with the
	// service shell-quoted in both slots.
	if got := c.restartCommand("nas"); got != "stopservice 'nas'; startservice 'nas'" {
		t.Errorf("restartCommand(nas) = %q", got)
	}
}

func TestRestartCommandCustomTemplate(t *testing.T) {
	c := NewClient(Config{Host: "10.0.0.1", RestartCommand: "service {service} restart"})
	if got := c.restartCommand("wan"); got != "service 'wan' restart" {
		t.Errorf("restartCommand(wan) custom = %q, want service 'wan' restart", got)
	}
	// Sentinels still force a full rc restart regardless of the template.
	if got := c.restartCommand("all"); got != fullRestart {
		t.Errorf("restartCommand(all) custom = %q, want %q", got, fullRestart)
	}
}

func TestConnectTimeoutSeconds(t *testing.T) {
	if got := connectTimeoutSeconds(0); got != 1 {
		t.Errorf("connectTimeoutSeconds(0) = %d, want 1 (floor)", got)
	}
	if got := connectTimeoutSeconds(30e9); got != 15 {
		t.Errorf("connectTimeoutSeconds(30s) = %d, want 15", got)
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(Config{Host: "10.0.0.1"})
	if c.user != "root" {
		t.Errorf("default user = %q, want root", c.user)
	}
	if c.sshBin != "ssh" {
		t.Errorf("default ssh binary = %q, want ssh", c.sshBin)
	}
	if c.addr != "10.0.0.1" || c.port != "" {
		t.Errorf("addr/port = %q/%q", c.addr, c.port)
	}
	if c.restartCmd != RestartCommandDefault {
		t.Errorf("default restart command = %q, want %q", c.restartCmd, RestartCommandDefault)
	}
}
