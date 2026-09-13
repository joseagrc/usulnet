package marketplace

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExposureOptionsValidate(t *testing.T) {
	e := &ExposureOptions{Domain: " WWW.APP.Example.com ", Service: " web ", Port: 8080}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	if e.Domain != "www.app.example.com" || e.Service != "web" {
		t.Fatalf("normalized exposure = %#v", e)
	}
	if got := e.domains(); got[0] != "app.example.com" || got[1] != "www.app.example.com" {
		t.Fatalf("domains = %#v", got)
	}
}

func TestExposureOptionsRejectsInvalidValues(t *testing.T) {
	for _, e := range []*ExposureOptions{
		{Domain: "https://example.com", Service: "web", Port: 80},
		{Domain: "example.com", Port: 80},
		{Domain: "example.com", Service: "web", Port: 0},
		{Domain: "example.com", Service: "web", Port: 65536},
	} {
		if err := e.Validate(); err == nil {
			t.Fatalf("expected validation error for %#v", e)
		}
	}
}

func TestApplyExposureToCompose(t *testing.T) {
	input := `services:
  db:
    image: postgres:16
  web:
    image: nginx:alpine
    networks: [internal]
    labels:
      keep: value
networks:
  internal: {}
`
	got, alias, err := ApplyExposureToCompose(input, "my-app", &ExposureOptions{
		Domain: "example.com", Service: "web", Port: 8080, WebSocket: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if alias != "my-app-web" {
		t.Fatalf("alias = %q", alias)
	}
	for _, want := range []string{"proxy-caddy", "usulnet.proxy.domain", "example.com", "my-app-web"} {
		if !strings.Contains(got, want) {
			t.Fatalf("transformed compose missing %q:\n%s", want, got)
		}
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("result is not YAML: %v", err)
	}
	services := parsed["services"].(map[string]any)
	db := services["db"].(map[string]any)
	if _, exists := db["networks"]; exists {
		t.Fatal("unselected service was modified")
	}
}

func TestApplyExposureRejectsMissingService(t *testing.T) {
	_, _, err := ApplyExposureToCompose("services:\n  web:\n    image: nginx\n", "app", &ExposureOptions{
		Domain: "example.com", Service: "api", Port: 80,
	})
	if err == nil {
		t.Fatal("expected missing service error")
	}
}
