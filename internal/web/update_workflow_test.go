// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2024-2026 usulnet contributors

package web

import (
	"context"
	"testing"
	"time"
)

type updateContextKey struct{}

type asyncUpdateCall struct {
	ctx           context.Context
	containerID   string
	backup        bool
	targetVersion string
}

// asyncUpdateService embeds the full interface because this test exercises
// only Apply; the remaining methods are intentionally never invoked.
type asyncUpdateService struct {
	UpdateService
	started chan asyncUpdateCall
}

func (s *asyncUpdateService) Apply(ctx context.Context, containerID string, backup bool, targetVersion string) error {
	s.started <- asyncUpdateCall{ctx: ctx, containerID: containerID, backup: backup, targetVersion: targetVersion}
	return nil
}

func TestStartContainerUpdateDetachesRequestCancellation(t *testing.T) {
	updates := &asyncUpdateService{started: make(chan asyncUpdateCall, 1)}
	h := &Handler{logger: &testLogger{}}

	requestCtx, cancel := context.WithCancel(context.WithValue(context.Background(), updateContextKey{}, "request-value"))
	h.startContainerUpdate(requestCtx, updates, "container-1", true, "1.2.3")
	cancel()

	select {
	case call := <-updates.started:
		if call.ctx.Err() != nil {
			t.Fatal("update context inherited request cancellation")
		}
		if got := call.ctx.Value(updateContextKey{}); got != "request-value" {
			t.Fatalf("request context value = %v, want request-value", got)
		}
		if call.containerID != "container-1" || !call.backup || call.targetVersion != "1.2.3" {
			t.Fatalf("unexpected update request: %+v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("update was not queued")
	}
}
