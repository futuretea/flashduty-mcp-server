package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/futuretea/flashduty-mcp-server/pkg/core/config"
	"github.com/futuretea/flashduty-mcp-server/pkg/core/logging"
	"github.com/futuretea/flashduty-mcp-server/pkg/core/version"
	internalhttp "github.com/futuretea/flashduty-mcp-server/pkg/server/http"
	"github.com/futuretea/flashduty-mcp-server/pkg/server/mcp"
)

// IOStreams represents standard input, output, and error streams
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// bindFlags binds command-line flags to viper configuration keys
func bindFlags(cmd *cobra.Command) {
	// Map of viper config key to flag name
	flagBindings := map[string]string{
		// Server configuration
		"port":         "port",
		"sse_base_url": "sse-base-url",
		"log_level":    "log-level",
		// FlashDuty API configuration
		"app_key":  "app-key",
		"base_url": "base-url",
		// Security configuration
		"read_only": "read-only",
		// Tool configuration
		"enabled_tools":  "enabled-tools",
		"disabled_tools": "disabled-tools",
	}

	for key, flag := range flagBindings {
		_ = viper.BindPFlag(key, cmd.Flags().Lookup(flag))
	}
}

// NewMCPServer creates a new cobra command for the FlashDuty MCP Server
func NewMCPServer(streams IOStreams) *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "flashduty-mcp-server",
		Short: "FlashDuty MCP Server - Model Context Protocol server for FlashDuty incident management",
		Long: `FlashDuty MCP Server exposes FlashDuty incident management over the
Model Context Protocol (MCP).

Run in stdio mode for direct MCP client integration, or in HTTP/SSE mode for
network access. Manage incidents, alerts, channels, teams, members, and
on-call schedules through the FlashDuty API.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			bindFlags(cmd)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cfgFile, streams)
		},
	}

	// Set output streams for the command
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)

	// Add configuration file flag
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (supports YAML)")

	// Server configuration flags
	cmd.Flags().Int("port", 0, "Port to listen on for HTTP/SSE mode (0 for stdio mode)")
	cmd.Flags().String("sse-base-url", "", "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	cmd.Flags().Int("log-level", 5, "Log level (0-9)")

	// FlashDuty API configuration flags
	cmd.Flags().String("app-key", "", "FlashDuty API app key")
	cmd.Flags().String("base-url", config.DefaultBaseURL, "FlashDuty API base URL")

	// Security configuration flags
	cmd.Flags().Bool("read-only", false, "Run in read-only mode (disables write operations)")

	// Tool configuration flags
	cmd.Flags().StringSlice("enabled-tools", []string{}, "Comma-separated list of tools to enable")
	cmd.Flags().StringSlice("disabled-tools", []string{}, "Comma-separated list of tools to disable")

	// Add version command
	cmd.AddCommand(newVersionCommand(streams))

	return cmd
}

// runServer runs the MCP server with the given configuration
func runServer(cfgFile string, streams IOStreams) error {
	// Load configuration from file, environment variables, and command-line flags
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logging early with configuration
	if cfg.Port == 0 {
		// Enable stdio mode - suppress all logging to avoid interfering with MCP protocol
		logging.SetStdioMode(true)
	} else {
		// HTTP/SSE mode - initialize normal logging
		logging.Initialize(cfg.LogLevel, streams.ErrOut)
	}

	// Create MCP server configuration
	mcpConfig := mcp.Configuration{
		StaticConfig: cfg,
	}

	// Create MCP server
	server, err := mcp.NewServer(mcpConfig)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}
	defer server.Close()

	// Start server based on port configuration
	if cfg.Port == 0 {
		// Stdio mode - use fmt.Fprintf for startup messages as logging is disabled
		fmt.Fprintf(streams.ErrOut, "Starting FlashDuty MCP Server in stdio mode\n")
		fmt.Fprintf(streams.ErrOut, "Enabled tools: %v\n", server.GetEnabledTools())
		return server.ServeStdio()
	}

	// HTTP/SSE mode - use logging
	logging.Info("Starting FlashDuty MCP Server in HTTP/SSE mode on port %d", cfg.Port)
	logging.Info("Enabled tools: %v", server.GetEnabledTools())
	if cfg.SSEBaseURL != "" {
		logging.Info("SSE Base URL: %s", cfg.SSEBaseURL)
	}

	ctx := context.Background()
	return internalhttp.Serve(ctx, server, cfg)
}

// newVersionCommand creates the version command
func newVersionCommand(streams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(streams.Out, "%s\n", version.GetVersionInfo())
		},
	}

	// Set output streams for the command
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)

	return cmd
}
