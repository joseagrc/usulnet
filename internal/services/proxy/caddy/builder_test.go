package caddy

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/fr4nsys/usulnet/internal/models"
)

func TestBuildConfigRedirectsNonCanonicalDomainToWWW(t *testing.T) {
	host := &models.ProxyHost{
		ID:                  uuid.New(),
		Enabled:             true,
		Domains:             []string{"example.com", "www.example.com"},
		UpstreamScheme:      models.ProxyUpstreamHTTP,
		UpstreamHost:        "app",
		UpstreamPort:        8080,
		SSLMode:             models.ProxySSLModeAuto,
		CanonicalWWWEnabled: true,
	}

	cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "", ":80", ":443")
	routes := cfg.Apps.HTTP.Servers["usulnet"].Routes
	if len(routes) != 2 {
		t.Fatalf("route count = %d, want 2", len(routes))
	}
	if got := routes[0].Match[0].Host; len(got) != 1 || got[0] != "example.com" {
		t.Fatalf("redirect hosts = %#v", got)
	}
	var redirect StaticResponseHandler
	if err := json.Unmarshal(routes[0].Handle[0], &redirect); err != nil {
		t.Fatalf("decode redirect: %v", err)
	}
	if redirect.StatusCode != 308 || redirect.Headers["Location"][0] != "https://www.example.com{http.request.uri}" {
		t.Fatalf("redirect = %#v", redirect)
	}
	if got := routes[1].Match[0].Host; len(got) != 1 || got[0] != "www.example.com" {
		t.Fatalf("canonical hosts = %#v", got)
	}
}

func TestBuildConfigProxiesBothDomainsBeforeCanonicalPromotion(t *testing.T) {
	host := &models.ProxyHost{
		ID: uuid.New(), Enabled: true, Domains: []string{"example.com", "www.example.com"},
		UpstreamScheme: models.ProxyUpstreamHTTP, UpstreamHost: "app", UpstreamPort: 80,
	}
	cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "", "", "")
	routes := cfg.Apps.HTTP.Servers["usulnet"].Routes
	if len(routes) != 1 {
		t.Fatalf("route count = %d, want one safe proxy route", len(routes))
	}
	if got := routes[0].Match[0].Host; len(got) != 2 || got[0] != "example.com" || got[1] != "www.example.com" {
		t.Fatalf("pre-promotion hosts = %#v", got)
	}
}

func TestBuildConfigLeavesSingleDomainAsProxy(t *testing.T) {
	host := &models.ProxyHost{
		ID: uuid.New(), Enabled: true, Domains: []string{"only.example.com"},
		UpstreamScheme: models.ProxyUpstreamHTTP, UpstreamHost: "app", UpstreamPort: 80,
	}
	cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "", "", "")
	if got := len(cfg.Apps.HTTP.Servers["usulnet"].Routes); got != 1 {
		t.Fatalf("route count = %d, want 1", got)
	}
}
