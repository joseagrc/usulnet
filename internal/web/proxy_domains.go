package web

import (
	"net"
	"strings"
)

// proxyDomains returns the hostnames that belong to a proxy host. A www alias
// is deliberately opt-in: APIs, private names, IP addresses and wildcard
// hostnames should not unexpectedly acquire a public TLS name.
func proxyDomains(domain string, includeWWW bool) []string {
	primary := strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if primary == "" {
		return nil
	}

	domains := []string{primary}
	if !includeWWW || !canHaveWWWAlias(primary) {
		return domains
	}

	return append(domains, "www."+primary)
}

func canHaveWWWAlias(domain string) bool {
	lower := strings.ToLower(domain)
	return !strings.HasPrefix(lower, "www.") &&
		!strings.Contains(lower, "*") &&
		net.ParseIP(lower) == nil &&
		strings.Contains(lower, ".")
}

func hasWWWAlias(domains []string, primary string) bool {
	wanted := "www." + strings.ToLower(strings.TrimSuffix(strings.TrimSpace(primary), "."))
	for _, domain := range domains {
		if strings.EqualFold(strings.TrimSuffix(strings.TrimSpace(domain), "."), wanted) {
			return true
		}
	}
	return false
}
