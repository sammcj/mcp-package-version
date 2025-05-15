package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
)

// registerRustTool registers the Rust version checking tool
func (s *PackageVersionServer) registerRustTool(srv *mcpserver.MCPServer) {
	// Create Rust handler with a logger
	rustHandler := handlers.NewRustHandler(s.logger, s.sharedCache)

	rustTool := mcp.NewTool("check_rust_versions",
		mcp.WithDescription("Get the current, up to date package versions to use when adding Rust crates or updating Cargo.toml"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Required: Dependencies from Cargo.toml"),
		),
	)

	// Add Rust handler
	srv.AddTool(rustTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_rust_versions").Debug("Received request")
		return rustHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}
