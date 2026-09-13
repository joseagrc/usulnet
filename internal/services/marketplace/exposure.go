package marketplace

import (
	"fmt"
	"regexp"
	"strings"
)

type ExposureOptions struct {
	Domain  string
	Service string
	Port    int
}

var exposureDomainPattern = regexp.MustCompile("^(?i)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$")

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
		return fmt.Errorf("%w: invalid exposure port", ErrInvalidInput)
	}
	return nil
}
