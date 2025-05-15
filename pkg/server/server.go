package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/internal/cache"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// CacheTTL is the time-to-live for cached data (12 hours)
	CacheTTL = 12 * time.Hour
	// MaxLogSize is the maximum size of the log file in megabytes before rotation
	MaxLogSize = 1
	// MaxLogBackups is the maximum number of old log files to retain
	MaxLogBackups = 3
	// MaxLogAge is the maximum number of days to retain old log files
	MaxLogAge = 28
)

// PackageVersionServer implements the MCPServerHandler interface for the package version server
type PackageVersionServer struct {
	logger      *logrus.Logger
	cache       *cache.Cache
	sharedCache *sync.Map
	Version     string
	Commit      string
	BuildDate   string
}

// getLogFilePath returns the path to the log file
func getLogFilePath() string {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home directory can't be determined
		return "mcp-package-version.log"
	}

	// Create logs directory in user's home directory if it doesn't exist
	logsDir := filepath.Join(homeDir, ".mcp-package-version", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		// Fallback to current directory if logs directory can't be created
		return "mcp-package-version.log"
	}

	return filepath.Join(logsDir, "mcp-package-version.log")
}

// NewPackageVersionServer creates a new package version server
func NewPackageVersionServer(version, commit, buildDate string) *PackageVersionServer {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level based on environment variable
	logLevelStr := os.Getenv("LOG_LEVEL")
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err == nil {
		logger.SetLevel(logLevel)
	} else {
		// Default to Info level if LOG_LEVEL is not set or invalid
		logger.SetLevel(logrus.InfoLevel)
	}
	logger.WithField("log_level", logger.GetLevel().String()).Debug("Log level set")

	logFilePath := getLogFilePath()

	// Configure log rotation
	logRotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    MaxLogSize,    // megabytes
		MaxBackups: MaxLogBackups, // number of backups
		MaxAge:     MaxLogAge,     // days
		Compress:   true,          // compress old log files
	}

	// Set logger output to the rotated log file initially
	// We will add stdout later only if transport is SSE
	logger.SetOutput(logRotator)

	// Create a fallback logger that discards all output in case we can't open the log file
	fallbackLogger := logrus.New()
	fallbackLogger.SetOutput(io.Discard)

	return &PackageVersionServer{
		logger:      logger,
		cache:       cache.NewCache(CacheTTL),
		sharedCache: &sync.Map{},
		Version:     version,
		Commit:      commit,
		BuildDate:   buildDate,
	}
}

// Name returns the display name of the server
func (s *PackageVersionServer) Name() string {
	return "Package Version"
}

// Capabilities returns the server capabilities
func (s *PackageVersionServer) Capabilities() []mcpserver.ServerOption {
	return []mcpserver.ServerOption{
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(false, false),
		mcpserver.WithPromptCapabilities(false),
	}
}

// Initialize sets up the server
func (s *PackageVersionServer) Initialize(srv *mcpserver.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Debug("Starting package-version MCP server")

	s.logger.Debug("Initialising package version handlers") // Register tools and handlers
	s.registerNpmTool(srv)
	s.registerPythonTools(srv)
	s.registerJavaTools(srv)
	s.registerGoTool(srv)
	s.registerRustTool(srv)
	s.registerDartTool(srv)
	s.registerBedrockTools(srv)
	s.registerDockerTool(srv)
	s.registerSwiftTool(srv)
	s.registerGitHubActionsTool(srv)

	// Register empty resource and prompt handlers to handle resources/list and prompts/list requests
	s.registerEmptyResourceHandlers(srv)
	s.registerEmptyPromptHandlers(srv)

	s.logger.Debug("All handlers registered successfully")

	return nil
}

// registerEmptyResourceHandlers registers empty resource handlers to respond to resources/list requests
func (s *PackageVersionServer) registerEmptyResourceHandlers(srv *mcpserver.MCPServer) {
	s.logger.Debug("Registering empty resource handlers")

	// The mcp-go library will automatically handle resources/list requests with an empty list
	// if no resources are registered, but we need to declare the capability
	// No need to add any actual resources since we don't have any
}

// registerEmptyPromptHandlers registers empty prompt handlers to respond to prompts/list requests
func (s *PackageVersionServer) registerEmptyPromptHandlers(srv *mcpserver.MCPServer) {
	s.logger.Debug("Registering empty prompt handlers")

	// The mcp-go library will automatically handle prompts/list requests with an empty list
	// if no prompts are registered, but we need to declare the capability
	// No need to add any actual prompts since we don't have any
}

// Start starts the MCP server with the specified transport
func (s *PackageVersionServer) Start(transport, port, baseURL string) error {
	s.logger.WithFields(logrus.Fields{
		"transport": transport,
		"port":      port,
		"baseURL":   baseURL,
	}).Debug("Starting MCP server")

	// Create a context with cancellation for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new server
	srv := mcpserver.NewMCPServer("package-version", "Package Version MCP Server")

	// Initialize the server
	if err := s.Initialize(srv); err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Run the server based on the transport type
	errCh := make(chan error, 1)

	if transport == "sse" {
		// Configure logger to also write to stdout for SSE mode
		logRotator := s.logger.Out.(*lumberjack.Logger) // Get the existing rotator
		multiWriter := io.MultiWriter(os.Stdout, logRotator)
		s.logger.SetOutput(multiWriter)
		s.logger.Debug("Configured logger for SSE mode (file + stdout)")

		// Create an SSE server
		// Ensure the baseURL has the correct format: http://hostname:port
		// Remove trailing slash if present
		if baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}

		// Ensure the baseURL is correctly formatted for SSE
		// The mcp-go package expects the baseURL to be in the format: http://hostname:port
		// without any trailing slashes or paths

		// First, check if baseURL already includes a port
		var sseBaseURL string
		if baseURL == "http://localhost" || baseURL == "https://localhost" {
			// If baseURL is just http://localhost or https://localhost, append the port
			sseBaseURL = fmt.Sprintf("%s:%s", baseURL, port)
		} else {
			// Otherwise, use the baseURL as is, assuming it already includes the port if needed
			// Otherwise, use the baseURL as is. It should contain the correct
			// scheme, hostname, and port (if non-standard) for external access.
			sseBaseURL = baseURL
		}

		// The --base-url provided by the user is assumed to be the correct external URL.
		// We no longer attempt to modify it or append the internal port, except for the localhost default case handled above.
		s.logger.WithField("final_advertised_base_url", sseBaseURL).Debug("Using final base URL for SSE configuration")

		// Create the SSE server with the correct base URL
		// The WithBaseURL option is critical for the client to connect properly
		// Try with different options to see what works

		// Try with a specific path for the SSE endpoint
		// The client might be expecting a specific path like /mcp/sse
		// Let's try with just the base URL without any path
		sseBaseURL = strings.TrimSuffix(sseBaseURL, "/mcp")

		// Add SSE server options
		sseOptions := []mcpserver.SSEOption{
			mcpserver.WithBaseURL(sseBaseURL),
			// Add any other relevant options here if discovered
		}
		s.logger.WithField("sse_options", fmt.Sprintf("%+v", sseOptions)).Debug("Configuring SSE server with options") // Log options

		// Create the SSE server with the options
		sseServer := mcpserver.NewSSEServer(srv, sseOptions...)

		// Start the SSE server in a goroutine
		go func() {
			// Start the SSE server on the specified port
			// The server will listen on all interfaces (0.0.0.0)
			listenAddr := ":" + port
			s.logger.WithFields(logrus.Fields{
				"listenAddr": listenAddr,
				"baseURL":    sseBaseURL,
				"serverName": "package-version",
			}).Info("Attempting to start SSE server") // Changed level to Info

			// Log the final configuration being used for SSE
			s.logger.WithFields(logrus.Fields{
				"listen_address":      listenAddr,
				"advertised_base_url": sseBaseURL,
			}).Info("SSE server configured")

			// Log the available routes for debugging
			s.logger.Debug("Expected SSE routes:")
			s.logger.Debug("- " + sseBaseURL + "/")
			s.logger.Debug("- " + sseBaseURL + "/sse")
			s.logger.Debug("- " + sseBaseURL + "/events")
			s.logger.Debug("- " + sseBaseURL + "/mcp")
			s.logger.Debug("- " + sseBaseURL + "/mcp/sse")

			// Try accessing the routes to see if they're available
			s.logger.Debug("Checking routes availability:")
			s.logger.Debug("To test routes, run: curl " + sseBaseURL + "/sse")

			if err := sseServer.Start(listenAddr); err != nil {
				// Log the error before sending it to the channel
				s.logger.WithError(err).Error("SSE server failed to start or encountered a runtime error")
				errCh <- fmt.Errorf("SSE server error: %w", err)
			} else {
				// This part might only be reached on graceful shutdown without error
				s.logger.Info("SSE server stopped gracefully")
			}
		}()

		// Wait for signal to shut down
		<-sigCh
		s.logger.Debug("Shutting down SSE server...")
		cancel()
		errCh <- nil
	} else {
		// Default to stdio transport
		go func() {

			s.logger.Debug("STDIO server is running. Press Ctrl+C to stop.")

			if err := mcpserver.ServeStdio(srv); err != nil {
				errCh <- fmt.Errorf("STDIO server error: %w", err)
			}
		}()

		// Wait for signal to shut down
		<-sigCh
		s.logger.Debug("Shutting down STDIO server...")
		cancel()
		errCh <- nil
	}

	// Wait for server to exit or error
	return <-errCh
}

// registerNpmTool registers the npm version checking tool
func (s *PackageVersionServer) registerNpmTool(srv *mcpserver.MCPServer) {
	// Create NPM handler with a logger that doesn't output to stdout/stderr in stdio mode
	npmHandler := handlers.NewNpmHandler(s.logger, s.sharedCache)

	// Add NPM tool
	npmTool := mcp.NewTool("check_npm_versions",
		mcp.WithDescription("Check latest stable versions for npm packages"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Required: Dependencies object from package.json (e.g., { \"dependencies\": { \"express\": \"^4.17.1\" } })"),
		),
		mcp.WithObject("constraints",
			mcp.Description("Optional constraints for specific packages"),
		),
	)

	// Add NPM handler
	srv.AddTool(npmTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_npm_versions").Debug("Received request")
		return npmHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerPythonTools registers the Python version checking tools
func (s *PackageVersionServer) registerPythonTools(srv *mcpserver.MCPServer) {
	// Create Python handler with a logger that doesn't output to stdout/stderr in stdio mode
	pythonHandler := handlers.NewPythonHandler(s.logger, s.sharedCache)

	// Tool for requirements.txt
	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Get the current, up to date Python package versions to use when adding or updating Python packages for requirements.txt"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Required: Array of one or more requirements from requirements.txt"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	// Add Python requirements.txt handler
	srv.AddTool(pythonTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_python_versions").Debug("Received request")
		return pythonHandler.GetLatestVersionFromRequirements(ctx, request.Params.Arguments)
	})

	// Tool for pyproject.toml
	pyprojectTool := mcp.NewTool("check_pyproject_versions",
		mcp.WithDescription("Get the current, up to date Python package versions to use when adding or updating Python packages for pyproject.toml"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Required: Dependencies object from pyproject.toml"),
		),
	)

	// Add Python pyproject.toml handler
	srv.AddTool(pyprojectTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_pyproject_versions").Debug("Received request")
		return pythonHandler.GetLatestVersionFromPyProject(ctx, request.Params.Arguments)
	})
}

// registerJavaTools registers the Java version checking tools
func (s *PackageVersionServer) registerJavaTools(srv *mcpserver.MCPServer) {
	// Create Java handler with a logger that doesn't output to stdout/stderr in stdio mode
	javaHandler := handlers.NewJavaHandler(s.logger, s.sharedCache)

	// Tool for Maven
	mavenTool := mcp.NewTool("check_maven_versions",
		mcp.WithDescription("Check latest stable versions for Java packages in pom.xml"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Maven dependencies"),
			mcp.Items(map[string]interface{}{"type": "object"}),
		),
	)

	// Add Maven handler
	srv.AddTool(mavenTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_maven_versions").Debug("Received request")
		return javaHandler.GetLatestVersionFromMaven(ctx, request.Params.Arguments)
	})

	// Tool for Gradle
	gradleTool := mcp.NewTool("check_gradle_versions",
		mcp.WithDescription("Get latest stable versions for Java packages in build.gradle"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Gradle dependencies"),
			mcp.Items(map[string]interface{}{"type": "object"}),
		),
	)

	// Add Gradle handler
	srv.AddTool(gradleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_gradle_versions").Debug("Received request")
		return javaHandler.GetLatestVersionFromGradle(ctx, request.Params.Arguments)
	})
}

// registerGoTool registers the Go version checking tool
func (s *PackageVersionServer) registerGoTool(srv *mcpserver.MCPServer) {
	// Create Go handler with a logger that doesn't output to stdout/stderr in stdio mode
	goHandler := handlers.NewGoHandler(s.logger, s.sharedCache)

	goTool := mcp.NewTool("check_go_versions",
		mcp.WithDescription("Get the current, up to date package versions to use when adding Go packages or updating go.mod"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Required: Dependencies from go.mod"),
		),
	)

	// Add Go handler
	srv.AddTool(goTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_go_versions").Debug("Received request")
		return goHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerBedrockTools registers the AWS Bedrock tools
func (s *PackageVersionServer) registerBedrockTools(srv *mcpserver.MCPServer) {
	// Create Bedrock handler with a logger that doesn't output to stdout/stderr in stdio mode
	bedrockHandler := handlers.NewBedrockHandler(s.logger, s.sharedCache)

	// Tool for searching Bedrock models
	bedrockTool := mcp.NewTool("check_bedrock_models",
		mcp.WithDescription("Search, list, and get information about Amazon Bedrock models"),
		mcp.WithString("action",
			mcp.Description("Action to perform: list all models, search for models, or get a specific model"),
			mcp.Enum("list", "search", "get"),
			mcp.DefaultString("list"),
		),
		mcp.WithString("query",
			mcp.Description("Search query for model name or ID (used with action: \"search\")"),
		),
		mcp.WithString("provider",
			mcp.Description("Filter by provider name (used with action: \"search\")"),
		),
		mcp.WithString("region",
			mcp.Description("Filter by AWS region (used with action: \"search\")"),
		),
		mcp.WithString("modelId",
			mcp.Description("Model ID to retrieve (used with action: \"get\")"),
		),
	)

	// Add Bedrock handler
	srv.AddTool(bedrockTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithFields(logrus.Fields{
			"tool":   "check_bedrock_models",
			"action": request.Params.Arguments["action"],
		}).Debug("Received request")
		return bedrockHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})

	// Tool for getting the latest Claude Sonnet model
	sonnetTool := mcp.NewTool("get_latest_bedrock_model",
		mcp.WithDescription("Return the latest Claude Sonnet model available on Amazon Bedrock (best for coding tasks)"),
	)

	// Add Bedrock Claude Sonnet handler
	srv.AddTool(sonnetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "get_latest_bedrock_model").Debug("Received request")
		// Set the action to get_latest_claude_sonnet to use the specialised method
		return bedrockHandler.GetLatestVersion(ctx, map[string]interface{}{
			"action": "get_latest_claude_sonnet",
		})
	})
}

// registerDockerTool registers the Docker version checking tool
func (s *PackageVersionServer) registerDockerTool(srv *mcpserver.MCPServer) {
	// Create Docker handler with a logger that doesn't output to stdout/stderr in stdio mode
	dockerHandler := handlers.NewDockerHandler(s.logger, s.sharedCache)

	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Get the latest, up to date tags for Docker container images from Docker Hub, GitHub Container Registry, or custom registries for use when writing Dockerfiles or docker-compose files"),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Required: Docker image name (e.g., \"nginx\", \"ubuntu\", \"ghcr.io/owner/repo\")"),
		),
		mcp.WithString("registry",
			mcp.Description("Registry to check (dockerhub, ghcr, or custom)"),
			mcp.Enum("dockerhub", "ghcr", "custom"),
			mcp.DefaultString("dockerhub"),
		),
		mcp.WithString("customRegistry",
			mcp.Description("URL for custom registry (required when registry is \"custom\")"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tags to return"),
			mcp.DefaultNumber(10),
		),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		// If the above doesn't work, maybe try a deeper structure like this:
		// mcp.WithArray("filterTags",
		//
		//	mcp.Description("Array of regex patterns to filter tags"),
		//	mcp.Items(map[string]interface{}{
		//
		// "type": "string",
		// "description": "Regex pattern to filter Docker tags",
		//
		//	}),
		//
		// ),
		mcp.WithBoolean("includeDigest",
			mcp.Description("Include image digest in results"),
			mcp.DefaultBool(false),
		),
	)

	// Add Docker handler
	srv.AddTool(dockerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithFields(logrus.Fields{
			"tool":     "check_docker_tags",
			"image":    request.Params.Arguments["image"],
			"registry": request.Params.Arguments["registry"],
		}).Debug("Received request")
		return dockerHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerSwiftTool registers the Swift version checking tool
func (s *PackageVersionServer) registerSwiftTool(srv *mcpserver.MCPServer) {
	// Create Swift handler with a logger that doesn't output to stdout/stderr in stdio mode
	swiftHandler := handlers.NewSwiftHandler(s.logger, s.sharedCache)

	swiftTool := mcp.NewTool("check_swift_versions",
		mcp.WithDescription("Check latest stable versions for Swift packages in Package.swift"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Required: Array of Swift package dependencies"),
			mcp.Items(map[string]interface{}{"type": "object"}),
		),
		mcp.WithObject("constraints",
			mcp.Description("Optional constraints for specific packages"),
		),
	)

	// Add Swift handler
	srv.AddTool(swiftTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_swift_versions").Debug("Received request")
		return swiftHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerGitHubActionsTool registers the GitHub Actions version checking tool
func (s *PackageVersionServer) registerGitHubActionsTool(srv *mcpserver.MCPServer) {
	// Create GitHub Actions handler with a logger that doesn't output to stdout/stderr in stdio mode
	githubActionsHandler := handlers.NewGitHubActionsHandler(s.logger, s.sharedCache)

	githubActionsTool := mcp.NewTool("check_github_actions",
		mcp.WithDescription("Get the current, up to date GitHub Actions versions to use when adding or updating GitHub Actions"),
		mcp.WithArray("actions",
			mcp.Required(),
			mcp.Description("Required: Array of GitHub Actions to check"),
			mcp.Items(map[string]interface{}{"type": "object"}),
		),
		mcp.WithBoolean("includeDetails",
			mcp.Description("Include additional details like published date and URL"),
			mcp.DefaultBool(false),
		),
	)

	// Add GitHub Actions handler
	srv.AddTool(githubActionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_github_actions").Debug("Received request")
		return githubActionsHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}
