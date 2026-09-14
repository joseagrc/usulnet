package caddy

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/fr4nsys/usulnet/internal/models"
)

func forceHTTPSHost() *models.ProxyHost {
	return &models.ProxyHost{
		ID:                uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Domains:           []string{"app.example.com"},
		Enabled:           true,
		UpstreamScheme:    models.ProxyUpstreamHTTP,
		UpstreamHost:      "app",
		UpstreamPort:      3000,
		SSLMode:           models.ProxySSLModeAuto,
		SSLForceHTTPS:     true,
		EnableWebSocket:   true,
		EnableCompression: true,
	}
}

func TestBuildConfigForceHTTPSAddsProtocolScopedRedirect(t *testing.T) {
	host := forceHTTPSHost()
	cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "admin@example.com", ":80", ":443")
	routes := cfg.Apps.HTTP.Servers["usulnet"].Routes

	if len(routes) != 2 {
		t.Fatalf("expected force-HTTPS and proxy routes, got %d", len(routes))
	}
	redirect := routes[0]
	if redirect.ID != "usulnet-11111111-1111-1111-1111-111111111111-force-https" {
		t.Fatalf("unexpected redirect route ID %q", redirect.ID)
	}
	if len(redirect.Match) != 1 || redirect.Match[0].Protocol != "http" {
		t.Fatalf("redirect must match only HTTP: %#v", redirect.Match)
	}
	if len(redirect.Match[0].Host) != 1 || redirect.Match[0].Host[0] != "app.example.com" {
		t.Fatalf("unexpected redirect hosts: %#v", redirect.Match[0].Host)
	}

	var handler StaticResponseHandler
	if err := json.Unmarshal(redirect.Handle[0], &handler); err != nil {
		t.Fatalf("decode redirect handler: %v", err)
	}
	if handler.StatusCode != 308 {
		t.Fatalf("expected 308, got %d", handler.StatusCode)
	}
	if got := handler.Headers["Location"]; len(got) != 1 || got[0] != "https://{http.request.host}{http.request.uri}" {
		t.Fatalf("unexpected redirect location: %#v", got)
	}

	if routes[1].Match[0].Protocol != "" {
		t.Fatalf("proxy route must continue matching both protocols: %#v", routes[1].Match)
	}
}

func TestBuildConfigDoesNotForceHTTPSWhenDisabledOrTLSIsNone(t *testing.T) {
	tests := []struct {
		name string
		edit func(*models.ProxyHost)
	}{
		{
			name: "flag disabled",
			edit: func(h *models.ProxyHost) { h.SSLForceHTTPS = false },
		},
		{
			name: "TLS disabled",
			edit: func(h *models.ProxyHost) { h.SSLMode = models.ProxySSLModeNone },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := forceHTTPSHost()
			tt.edit(host)
			cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "", ":80", ":443")
			routes := cfg.Apps.HTTP.Servers["usulnet"].Routes
			if len(routes) != 1 {
				t.Fatalf("expected only proxy route, got %d", len(routes))
			}
			if routes[0].Match[0].Protocol != "" {
				t.Fatalf("unexpected protocol matcher: %#v", routes[0].Match)
			}
		})
	}
}

func TestBuildConfigForceHTTPSRunsBeforeCanonicalWWWRedirect(t *testing.T) {
	host := forceHTTPSHost()
	host.Domains = []string{"example.com", "www.example.com"}
	host.CanonicalWWWEnabled = true

	cfg := BuildConfig([]*models.ProxyHost{host}, nil, nil, "", ":80", ":443")
	routes := cfg.Apps.HTTP.Servers["usulnet"].Routes
	if len(routes) != 3 {
		t.Fatalf("expected force-HTTPS, canonical, and proxy routes, got %d", len(routes))
	}
	if routes[0].Match[0].Protocol != "http" {
		t.Fatalf("force-HTTPS route must be first: %#v", routes)
	}
	if routes[1].ID != "usulnet-11111111-1111-1111-1111-111111111111-canonical-redirect" {
		t.Fatalf("canonical redirect must remain second, got %q", routes[1].ID)
	}
}
