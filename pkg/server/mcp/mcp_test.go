package mcp
package mcp

import (
	"testing"

	"github.com/futuretea/flashduty-mcp-server/pkg/core/config"
)

func newTestServer(enabledTools, disabledTools []string) *Server {
	return &Server{
		configuration: &Configuration{
			StaticConfig: &config.StaticConfig{
				EnabledTools:  enabledTools,
				DisabledTools: disabledTools,
			},
		},
	}
}

func TestShouldEnableTool_AllEnabled(t *testing.T) {
	s := newTestServer(nil, nil)
	if !s.shouldEnableTool("list_incidents") {
		t.Error("expected tool to be enabled when no filters are set")
	}
	if !s.shouldEnableTool("any_tool_name") {
		t.Error("expected any tool to be enabled when no filters are set")
	}
}

func TestShouldEnableTool_EnabledList(t *testing.T) {
	s := newTestServer([]string{"list_incidents", "get_incident"}, nil)

	if !s.shouldEnableTool("list_incidents") {
		t.Error("list_incidents should be enabled")
	}
	if !s.shouldEnableTool("get_incident") {
		t.Error("get_incident should be enabled")
	}
	if s.shouldEnableTool("create_incident") {
		t.Error("create_incident should NOT be enabled (not in enabled list)")
	}
}

func TestShouldEnableTool_DisabledList(t *testing.T) {
	s := newTestServer(nil, []string{"create_incident", "resolve_incidents"})

	if !s.shouldEnableTool("list_incidents") {
		t.Error("list_incidents should be enabled (not in disabled list)")
	}
	if s.shouldEnableTool("create_incident") {
		t.Error("create_incident should be disabled")
	}
	if s.shouldEnableTool("resolve_incidents") {
		t.Error("resolve_incidents should be disabled")
	}
}

func TestShouldEnableTool_DisabledTakesPriority(t *testing.T) {
	// Tool is in both enabled and disabled lists - disabled should win
	s := newTestServer(
		[]string{"list_incidents", "create_incident"},
		[]string{"create_incident"},
	)

	if !s.shouldEnableTool("list_incidents") {
		t.Error("list_incidents should be enabled")
	}
	if s.shouldEnableTool("create_incident") {
		t.Error("create_incident should be disabled (disabled takes priority)")
	}
}

func TestShouldEnableTool_EmptySlices(t *testing.T) {
	s := newTestServer([]string{}, []string{})
	if !s.shouldEnableTool("list_incidents") {
		t.Error("expected tool to be enabled with empty slices")
	}
}
