// SPDX-License-Identifier: AGPL-3.0-or-later
package marketplace

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const proxyNetworkName = "proxy-caddy"

// ExposureOptions describes an optional public endpoint for a Marketplace app.
// Service and Port are explicit to avoid exposing an unintended container.
type ExposureOptions struct {
	Domain    string
	Service   string
	Port      int
	WebSocket bool
}

var (
	exposureDomainPattern = regexp.MustCompile(`(?i)^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
	aliasSanitizer        = regexp.MustCompile(`[^a-z0-9-]+`)
)

func (e *ExposureOptions) Validate() error {
	if e == nil {
		return nil
	}
	e.Domain = strings.ToLower(strings.TrimSpace(e.Domain))
	e.Service = strings.TrimSpace(e.Service)
	if !exposureDomainPattern.MatchString(e.Domain) {
		return fmt.Errorf("%w: invalid exposure domain", ErrInvalidInput)
	}
	if e.Service == "" {
		return fmt.Errorf("%w: exposure service is required", ErrInvalidInput)
	}
	if e.Port < 1 || e.Port > 65535 {
		return fmt.Errorf("%w: exposure port must be between 1 and 65535", ErrInvalidInput)
	}
	return nil
}

func (e *ExposureOptions) domains() []string {
	root := strings.TrimPrefix(e.Domain, "www.")
	return []string{root, "www." + root}
}

func exposureAlias(stackName, service string) string {
	alias := strings.ToLower(stackName + "-" + service)
	return strings.Trim(aliasSanitizer.ReplaceAllString(alias, "-"), "-")
}

// ApplyExposureToCompose connects one service to proxy-caddy and adds the
// standard discovery labels, preserving all unrelated Compose fields.
func ApplyExposureToCompose(body, stackName string, exposure *ExposureOptions) (string, string, error) {
	if exposure == nil {
		return body, "", nil
	}
	if err := exposure.Validate(); err != nil {
		return "", "", err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
		return "", "", fmt.Errorf("%w: parse compose: %v", ErrInvalidInput, err)
	}
	root := documentMap(&doc)
	services := mapValue(root, "services")
	if root == nil || services == nil || services.Kind != yaml.MappingNode {
		return "", "", fmt.Errorf("%w: compose services are required", ErrInvalidInput)
	}
	service := mapValue(services, exposure.Service)
	if service == nil || service.Kind != yaml.MappingNode {
		return "", "", fmt.Errorf("%w: exposure service %q does not exist", ErrInvalidInput, exposure.Service)
	}
	alias := exposureAlias(stackName, exposure.Service)
	if alias == "" {
		return "", "", fmt.Errorf("%w: invalid stack/service alias", ErrInvalidInput)
	}
	ensureTopLevelNetwork(root)
	ensureServiceNetwork(service, alias)
	ensureLabels(service, map[string]string{
		"usulnet.proxy.domain":    exposure.Domain,
		"usulnet.proxy.port":      strconv.Itoa(exposure.Port),
		"usulnet.proxy.ssl":       "true",
		"usulnet.proxy.websocket": strconv.FormatBool(exposure.WebSocket),
	})
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return "", "", fmt.Errorf("marshal compose exposure: %w", err)
	}
	return string(out), alias, nil
}

func documentMap(doc *yaml.Node) *yaml.Node {
	if doc == nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	return doc.Content[0]
}

func mapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func setMapValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, scalar(key), value)
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func ensureTopLevelNetwork(root *yaml.Node) {
	networks := mapValue(root, "networks")
	if networks == nil || networks.Kind != yaml.MappingNode {
		networks = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapValue(root, "networks", networks)
	}
	proxy := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMapValue(proxy, "external", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	setMapValue(proxy, "name", scalar(proxyNetworkName))
	setMapValue(networks, proxyNetworkName, proxy)
}

func ensureServiceNetwork(service *yaml.Node, alias string) {
	networks := mapValue(service, "networks")
	if networks == nil {
		networks = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapValue(service, "networks", networks)
	} else if networks.Kind == yaml.SequenceNode {
		converted := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, entry := range networks.Content {
			setMapValue(converted, entry.Value, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"})
		}
		networks = converted
		setMapValue(service, "networks", networks)
	}
	proxy := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMapValue(proxy, "aliases", &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{scalar(alias)}})
	setMapValue(networks, proxyNetworkName, proxy)
}

func ensureLabels(service *yaml.Node, labels map[string]string) {
	current := mapValue(service, "labels")
	if current == nil {
		current = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapValue(service, "labels", current)
	}
	if current.Kind == yaml.SequenceNode {
		for key, value := range labels {
			current.Content = append(current.Content, scalar(key+"="+value))
		}
		return
	}
	for key, value := range labels {
		setMapValue(current, key, scalar(value))
	}
}

func waitForTCP(ctx context.Context, address string) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", address)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return fmt.Errorf("upstream %s did not become reachable: %w", address, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func validateExposureDNS(ctx context.Context, domains []string) error {
	for _, domain := range domains {
		addresses, err := net.DefaultResolver.LookupHost(ctx, domain)
		if err != nil || len(addresses) == 0 {
			return fmt.Errorf("DNS for %s is not ready: %w", domain, err)
		}
	}
	return nil
}

// waitForPublicEndpoint verifies both the canonical redirect and a trusted TLS
// connection to www. Any HTTP response from the upstream is acceptable; TLS
// and routing, rather than application authentication, are being tested.
func waitForPublicEndpoint(ctx context.Context, domains []string) error {
	if len(domains) != 2 {
		return fmt.Errorf("canonical domain pair is required")
	}
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domains[0]+"/", nil)
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			location := response.Header.Get("Location")
			if response.StatusCode == http.StatusPermanentRedirect && strings.HasPrefix(location, "https://"+domains[1]) {
				canonical, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domains[1]+"/", nil)
				canonicalResponse, canonicalErr := client.Do(canonical)
				if canonicalErr == nil {
					_ = canonicalResponse.Body.Close()
					return nil
				}
				lastErr = canonicalErr
			} else {
				lastErr = fmt.Errorf("root returned %d with location %q", response.StatusCode, location)
			}
		} else {
			lastErr = err
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return fmt.Errorf("public endpoint did not become ready: %w", lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}
