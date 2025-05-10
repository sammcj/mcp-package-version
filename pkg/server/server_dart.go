package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
)

// registerDartTool registers the Dart version checking tool
func (s *PackageVersionServer) registerDartTool(srv *mcpserver.MCPServer) {
	// Create Dart handler with a logger
	dartHandler := handlers.NewDartHandler(s.logger, s.sharedCache)

	dartTool := mcp.NewTool("check_dart_versions",
		mcp.WithDescription("Get the current, up to date package versions to use when adding Dart packages or updating pubspec.yaml"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Required: Dependencies from pubspec.yaml"),
		),
	)

	// Add Dart handler
	srv.AddTool(dartTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_dart_versions").Debug("Received request")
		return dartHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}
