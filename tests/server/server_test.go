package server_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestToolSchemaValidation tests that all tool schemas can be validated
// and match the MCP specification requirements
func TestToolSchemaValidation(t *testing.T) {
	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := &server.PackageVersionServer{
		Logger:      logger,
		SharedCache: &sync.Map{},
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
	// Note: Since GetTools() doesn't exist, we need another way to access tools
	// For now we'll skip this test since we can't access the tools directly
	t.Skip("Skipping test since we can't access tools directly from the MCP server")
	
	// The following code would be ideal if we had access to tools:
	/*
	tools := srv.GetRegisteredTools() // hypothetical method
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
	*/
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

// TestAllArrayParameters tests tools with array parameters
func TestAllArrayParameters(t *testing.T) {
	// Since we can't access tools directly, we'll test individual handlers
	// in their respective test files instead of from the server
	t.Skip("Testing array parameters in individual handler tests instead")
}