// Package mcp configures the FlashDuty MCP server and its tools.
package mcp

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/futuretea/flashduty-mcp-server/pkg/core/config"
	"github.com/futuretea/flashduty-mcp-server/pkg/core/logging"
	"github.com/futuretea/flashduty-mcp-server/pkg/core/version"
	"github.com/futuretea/flashduty-mcp-server/pkg/toolset"
	flashdutyToolset "github.com/futuretea/flashduty-mcp-server/pkg/toolset/flashduty"
)

// Configuration holds the server's static settings.
type Configuration struct {
	*config.StaticConfig
}

// Server represents the MCP server
type Server struct {
	configuration *Configuration
	server        *server.MCPServer
	enabledTools  []string
	client        *flashdutyToolset.Client
}

// NewServer creates a new MCP server with the given configuration
func NewServer(configuration Configuration) (*Server, error) {
	var serverOptions []server.ServerOption

	// Configure server capabilities
	serverOptions = append(serverOptions,
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Initialize FlashDuty client
	client := flashdutyToolset.NewClient(
		configuration.BaseURL,
		configuration.AppKey,
		configuration.DefaultTimezone,
		flashdutyToolset.WithInsecureSkipTLSVerify(configuration.InsecureSkipTLSVerify),
	)
	logging.Info("FlashDuty client initialized (base_url: %s)", configuration.BaseURL)

	s := &Server{
		configuration: &configuration,
		server:        server.NewMCPServer(version.BinaryName, version.Version, serverOptions...),
		client:        client,
	}

	// Register tools
	if err := s.registerTools(); err != nil {
		return nil, err
	}

	return s, nil
}

// registerTools registers all available tools based on configuration
func (s *Server) registerTools() error {
	// Initialize toolsets
	fdTs := &flashdutyToolset.Toolset{
		ReadOnly: s.configuration.ReadOnly,
	}

	// Get tools from the toolset
	tools := fdTs.GetTools(s.client)

	// Register tools based on configuration
	for _, tool := range tools {
		if s.shouldEnableTool(tool.Tool.Name) {
			if err := s.registerTool(tool); err != nil {
				return fmt.Errorf("failed to register tool %s: %w", tool.Tool.Name, err)
			}
		}
	}

	logging.Info("MCP server initialized with %d tools", len(s.enabledTools))
	return nil
}

// shouldEnableTool determines if a tool should be enabled based on configuration.
func (s *Server) shouldEnableTool(toolName string) bool {
	if slices.Contains(s.configuration.DisabledTools, toolName) {
		return false
	}
	if len(s.configuration.EnabledTools) > 0 {
		return slices.Contains(s.configuration.EnabledTools, toolName)
	}
	return true
}

// registerTool registers a single tool with the MCP server
func (s *Server) registerTool(tool toolset.ServerTool) error {
	client := s.client

	toolHandler := server.ToolHandlerFunc(func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.Debug("Tool %s called with params: %v", tool.Tool.Name, request.Params.Arguments)

		params, _ := request.Params.Arguments.(map[string]any)
		if params == nil {
			params = make(map[string]any)
		}

		result, err := tool.Handler(client, params)
		return NewTextResult(result, err), nil
	})

	// Register tool with the MCP server
	s.server.AddTool(tool.Tool, toolHandler)
	s.enabledTools = append(s.enabledTools, tool.Tool.Name)

	logging.Info("Registered tool: %s", tool.Tool.Name)
	return nil
}

// ServeStdio starts the MCP server in stdio mode
func (s *Server) ServeStdio() error {
	logging.Info("Starting MCP server in stdio mode")
	return server.ServeStdio(s.server)
}

// ServeSse starts the MCP server in SSE mode
func (s *Server) ServeSse(baseURL string, httpServer *http.Server) *server.SSEServer {
	logging.Info("Starting MCP server in SSE mode")

	options := []server.SSEOption{
		server.WithHTTPServer(httpServer),
	}

	if baseURL != "" {
		options = append(options, server.WithBaseURL(baseURL))
	}

	return server.NewSSEServer(s.server, options...)
}

// ServeHTTP starts the MCP server in HTTP mode
func (s *Server) ServeHTTP(httpServer *http.Server) *server.StreamableHTTPServer {
	logging.Info("Starting MCP server in HTTP mode")

	options := []server.StreamableHTTPOption{
		server.WithStreamableHTTPServer(httpServer),
		server.WithStateLess(true),
	}

	return server.NewStreamableHTTPServer(s.server, options...)
}

// GetEnabledTools returns the list of enabled tools
func (s *Server) GetEnabledTools() []string {
	return s.enabledTools
}

// IsHealthy returns true if the server is properly initialized
func (s *Server) IsHealthy() bool {
	return s.client != nil
}

// Close cleans up the server resources
func (s *Server) Close() {
	logging.Info("Closing MCP server")
}

// NewTextResult creates a standardized text result for tool responses.
func NewTextResult(content string, err error) *mcp.CallToolResult {
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: err.Error()},
			},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: content},
		},
	}
}
