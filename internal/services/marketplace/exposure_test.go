package marketplace

import "testing"

func TestExposureOptionsValidate(t *testing.T) {
	e := &ExposureOptions{Domain: " APP.example.com ", Service: "web", Port: 8080}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	if e.Domain != "app.example.com" {
		t.Fatalf("domain=%q", e.Domain)
	}
}
