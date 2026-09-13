// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2024-2026 usulnet contributors

package update

import (
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func TestEffectiveHealthCheckWait(t *testing.T) {
	tests := []struct {
		name       string
		configured time.Duration
		hasProbe   bool
		want       time.Duration
	}{
		{"extends a short Docker healthcheck window", 30 * time.Second, true, minimumHealthCheckWait},
		{"keeps an explicitly sufficient window", 2 * time.Minute, true, 2 * time.Minute},
		{"does not delay containers without a healthcheck", 30 * time.Second, false, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &dockertypes.ContainerJSON{Config: &container.Config{}}
			if tt.hasProbe {
				info.Config.Healthcheck = &container.HealthConfig{Test: []string{"CMD", "true"}}
			}

			if got := effectiveHealthCheckWait(tt.configured, info); got != tt.want {
				t.Fatalf("effectiveHealthCheckWait() = %s, want %s", got, tt.want)
			}
		})
	}
}
