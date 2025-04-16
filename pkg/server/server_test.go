package server

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestToolSchemaValidation tests that all tool schemas can be validated
// and match the MCP specification requirements
func TestToolSchemaValidation(t *testing.T) {
	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := &PackageVersionServer{
		logger:      logger,
		sharedCache: &map[string]interface{}{},
		Version:     "test",
		Commit:      "test",
		BuildDate:   "test",
	}

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Server")

	// Initialize the server, which registers all tools
	err := s.Initialize(srv)
	assert.NoError(t, err, "Server initialization should not fail")

	// Get all registered tools
	tools := srv.GetTools()
	assert.NotEmpty(t, tools, "Server should have registered tools")

	// Test each tool's schema for validity
	for _, tool := range tools {
		t.Run("Tool_"+tool.Name, func(t *testing.T) {
			// Check that tool has a valid name
			assert.NotEmpty(t, tool.Name, "Tool name should not be empty")

			// Check that the tool has a description
			assert.NotEmpty(t, tool.Description, "Tool description should not be empty")

			// Validate the input schema
			validateToolInputSchema(t, tool)
		})
	}
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

// TestDockerToolSchema specifically tests the Docker tool's schema
// since this was mentioned as problematic with AWS validation
func TestDockerToolSchema(t *testing.T) {
	// Create a mock server to register the Docker tool
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	server := &PackageVersionServer{
		logger:      logger,
		sharedCache: &map[string]interface{}{},
	}

	srv := mcpserver.NewMCPServer("test-server", "Test Server")

	// Register just the Docker tool
	server.registerDockerTool(srv)

	// Get the registered Docker tool
	tools := srv.GetTools()
	assert.NotEmpty(t, tools, "Server should have registered the Docker tool")

	var dockerTool *mcp.Tool
	for _, tool := range tools {
		if tool.Name == "check_docker_tags" {
			dockerTool = &tool
			break
		}
	}

	assert.NotNil(t, dockerTool, "Docker tool should be registered")

	if dockerTool == nil {
		t.FailNow()
	}

	// Convert the schema to JSON for examination
	schemaJSON, err := json.Marshal(dockerTool.InputSchema)
	assert.NoError(t, err, "Schema should be marshallable to JSON")

	// Parse the schema back as a map for examination
	var schema map[string]interface{}
	err = json.Unmarshal(schemaJSON, &schema)
	assert.NoError(t, err, "Schema should be unmarshallable from JSON")

	// Check the properties
	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok, "Schema should have properties")

	// Check the filterTags array parameter specifically
	filterTags, ok := properties["filterTags"].(map[string]interface{})
	assert.True(t, ok, "Schema should have filterTags property")

	// Verify filterTags is an array
	propType, hasType := filterTags["type"]
	assert.True(t, hasType, "filterTags should have a type")
	assert.Equal(t, "array", propType, "filterTags should be an array")

	// Verify filterTags has items property
	items, hasItems := filterTags["items"]
	assert.True(t, hasItems, "filterTags must have items defined")
	assert.NotNil(t, items, "filterTags items must not be null")

	// Check the items property structure
	itemsMap, ok := items.(map[string]interface{})
	assert.True(t, ok, "filterTags items must be a valid object")
	assert.Equal(t, "string", itemsMap["type"], "filterTags items should be of type string")
}

// TestAllArrayParameters tests all tools with array parameters
// to ensure they conform to MCP specification
func TestAllArrayParameters(t *testing.T) {
	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := &PackageVersionServer{
		logger:      logger,
		sharedCache: &map[string]interface{}{},
		Version:     "test",
		Commit:      "test",
		BuildDate:   "test",
	}

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Server")

	// Initialize the server, which registers all tools
	s.Initialize(srv)

	// Define the tools with array parameters that we want to check
	toolsWithArrayParams := map[string][]string{
		"check_docker_tags":   {"filterTags"},
		"check_python_versions": {"requirements"},
		"check_maven_versions": {"dependencies"},
		"check_gradle_versions": {"dependencies"},
		"check_swift_versions": {"dependencies"},
		"check_github_actions": {"actions"},
	}

	// Get all registered tools
	tools := srv.GetTools()

	// Test each specified tool
	for toolName, arrayParams := range toolsWithArrayParams {
		t.Run(toolName, func(t *testing.T) {
			// Find the tool
			var tool *mcp.Tool
			for _, t := range tools {
				if t.Name == toolName {
					tool = &t
					break
				}
			}

			assert.NotNil(t, tool, "Tool %s should be registered", toolName)
			if tool == nil {
				return
			}

			// Convert the schema to JSON for examination
			schemaJSON, err := json.Marshal(tool.InputSchema)
			assert.NoError(t, err, "Schema should be marshallable to JSON")

			// Parse the schema back as a map for examination
			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			assert.NoError(t, err, "Schema should be unmarshallable from JSON")

			// Check the properties
			properties, ok := schema["properties"].(map[string]interface{})
			assert.True(t, ok, "Schema should have properties")

			// Check each array parameter
			for _, paramName := range arrayParams {
				param, ok := properties[paramName].(map[string]interface{})
				assert.True(t, ok, "Schema should have %s property", paramName)

				// Verify it's an array
				propType, hasType := param["type"]
				assert.True(t, hasType, "%s should have a type", paramName)
				assert.Equal(t, "array", propType, "%s should be an array", paramName)

				// Verify it has items property
				items, hasItems := param["items"]
				assert.True(t, hasItems, "%s must have items defined", paramName)
				assert.NotNil(t, items, "%s items must not be null", paramName)

				// Check the items property structure
				itemsMap, ok := items.(map[string]interface{})
				assert.True(t, ok, "%s items must be a valid object", paramName)
				assert.NotEmpty(t, itemsMap["type"], "%s items must have a type", paramName)
			}
		})
	}
}
