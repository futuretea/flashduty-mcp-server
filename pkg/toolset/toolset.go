package toolset

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Toolset defines the interface for a set of MCP tools.
type Toolset interface {
	// GetName returns the name of the toolset.
	GetName() string

	// GetDescription returns the description of the toolset.
	GetDescription() string

	// GetTools returns the tools provided by this toolset.
	GetTools(client any) []ServerTool
}

// ServerTool represents an MCP tool with its metadata and handler.
type ServerTool struct {
	// Tool is the MCP tool definition.
	Tool mcp.Tool

	// Handler is the function that handles tool calls.
	Handler ToolHandler
}

// ToolHandler is the function signature for handling tool calls.
type ToolHandler func(client any, params map[string]any) (string, error)
