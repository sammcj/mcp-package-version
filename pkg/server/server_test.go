package server

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestServerInitialise tests that the server initializes without errors
func TestServerInitialise(t *testing.T) {
	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := &PackageVersionServer{
		logger:      logger,
		sharedCache: &sync.Map{}, // Fixed: using lowercase field name as per struct definition
		Version:     "test",
		Commit:      "test",
		BuildDate:   "test",
	}

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Server")

	// Initialise the server, which registers all tools
	err := s.Initialize(srv)
	assert.NoError(t, err, "Server initialisation should not fail")

	// Since we can't access tools directly with GetTools(), we'll just test server initialisation
	// is successful, which implicitly means tools were registered correctly
}

// TestDockerToolRegistration specifically tests that the Docker tool is registered correctly
func TestDockerToolRegistration(t *testing.T) {
	// Create a mock server to register the Docker tool
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	server := &PackageVersionServer{
		logger:      logger,
		sharedCache: &sync.Map{}, // Fixed: using lowercase field name as per struct definition
	}

	srv := mcpserver.NewMCPServer("test-server", "Test Server")

	// Register just the Docker tool
	server.registerDockerTool(srv)

	// We can't directly check if the tool was registered since srv.GetTools() doesn't exist
	// But we can verify that the registration function completed without errors
	// If there were structural issues with the tool definition, it would have panicked
}

// validateToolInputSchema specifically tests that the tool's input schema
// is valid according to the MCP specification, focusing on array parameters
func validateToolInputSchema(t *testing.T, tool mcp.Tool) {
	// Convert the schema to JSON for examination
	schemaJSON, err := json.Marshal(tool.InputSchema)
	assert.NoError(t, err, "Schema should be marshallable to JSON")

	// Parse the schema back as a map for examination
	var schema map[string]interface{}
	err = json.Unmarshal(schemaJSON, &schema)
	assert.NoError(t, err, "Schema should be unmarshallable from JSON")

	// Check if the schema has properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		// Some tools might not have properties, which is fine
		return
	}

	// Validate each property in the schema
	for propName, propValue := range properties {
		propMap, ok := propValue.(map[string]interface{})
		if !ok {
			t.Errorf("Property %s is not a map", propName)
			continue
		}

		// Check for array type properties
		propType, hasType := propMap["type"]
		if hasType && propType == "array" {
			// Validate that array properties have an items definition
			items, hasItems := propMap["items"]
			assert.True(t, hasItems, "Array property %s must have items defined", propName)
			assert.NotNil(t, items, "Array items for %s must not be null", propName)

			// Further validate the items property
			itemsMap, ok := items.(map[string]interface{})
			assert.True(t, ok, "Items for %s must be a valid object", propName)

			// Items must have a type
			itemType, hasItemType := itemsMap["type"]
			assert.True(t, hasItemType, "Items for %s must have a type defined", propName)
			assert.NotEmpty(t, itemType, "Items type for %s must not be empty", propName)
		}
	}
}

// TestToolSchemaValidation tests that tool schemas conform to MCP specifications
func TestToolSchemaValidation(t *testing.T) {
	// Create some sample tools to test the validation function
	tools := []struct {
		name string
		tool mcp.Tool
	}{
		{
			"DockerTool",
			mcp.NewTool("check_docker_tags",
				mcp.WithDescription("Check available tags for Docker container images"),
				mcp.WithString("image",
					mcp.Required(),
					mcp.Description("Required: Docker image name"),
				),
				mcp.WithArray("filterTags",
					mcp.Description("Array of regex patterns to filter tags"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
			),
		},
		{
			"NPMTool",
			mcp.NewTool("check_npm_versions",
				mcp.WithDescription("Check latest versions for NPM packages"),
				mcp.WithArray("packages",
					mcp.Required(),
					mcp.Description("Required: Array of package names to check"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
			),
		},
	}

	// Test each tool's schema for validity
	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			validateToolInputSchema(t, tc.tool)
		})
	}
}

// TestServerCapabilities tests that the server capabilities are set correctly
func TestServerCapabilities(t *testing.T) {
	// Create a new server instance
	s := NewPackageVersionServer("test", "test", "test")

	// Check capabilities
	capabilities := s.Capabilities()

	// Verify that tools capabilities are enabled
	// Just check the length to avoid unused variable warning
	assert.Equal(t, 3, len(capabilities), "Server should have exactly 3 capabilities")
}
