package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
)

// registerComposerTool registers the Composer/Laravel package version checking tool
func (s *PackageVersionServer) registerComposerTool(srv *server.MCPServer) {
	composerHandler := handlers.NewComposerHandler(s.logger, s.sharedCache)

	composerTool := mcp.NewTool("check_composer_versions",
		mcp.WithDescription("Check latest stable versions for Laravel/Composer packages"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Dependencies object from composer.json"),
		),
		mcp.WithObject("constraints",
			mcp.Description("Optional constraints for specific packages"),
		),
	)

	// Add Composer handler
	srv.AddTool(composerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.Info(fmt.Sprintf("Received composer version check request: %+v", request.Params.Arguments))

		result, err := composerHandler.GetLatestVersion(ctx, request.Params.Arguments)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Error processing composer version check: %v", err))
			return nil, err
		}

		if result == nil {
			s.logger.Error("Composer handler returned nil result")
			return nil, fmt.Errorf("handler returned nil result")
		}

		s.logger.Info(fmt.Sprintf("Composer version check completed: %+v", result))
		return result, nil
	})
}
