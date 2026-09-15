package web

import "testing"

func TestProxyDomainsAddsOptInWWWAlias(t *testing.T) {
	got := proxyDomains("server1.joseagrc.com", true)
	if len(got) != 2 || got[0] != "server1.joseagrc.com" || got[1] != "www.server1.joseagrc.com" {
		t.Fatalf("unexpected domains: %#v", got)
	}
}

func TestProxyDomainsSkipsInvalidWWWAliases(t *testing.T) {
	for _, domain := range []string{"www.example.com", "127.0.0.1", "*.example.com", "localhost"} {
		got := proxyDomains(domain, true)
		if len(got) != 1 || got[0] != domain {
			t.Errorf("%q: unexpected domains %#v", domain, got)
		}
	}
}

func TestHasWWWAlias(t *testing.T) {
	if !hasWWWAlias([]string{"server1.joseagrc.com", "www.server1.joseagrc.com"}, "server1.joseagrc.com") {
		t.Fatal("expected www alias to be detected")
	}
}
