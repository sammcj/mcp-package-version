package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPSchemaCompliance validates that all tools registered by the server
// comply with the MCP schema specification, particularly focusing on array parameters
// which need to have properly defined 'items' properties.
func TestMCPSchemaCompliance(t *testing.T) {
	// Create a new server instance
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	err := s.Initialize(srv)
	require.NoError(t, err, "Server initialization should not fail")

	// Get all registered tools
	tools := srv.GetTools()
	require.NotEmpty(t, tools, "Server should have registered some tools")

	// The original AWS error mentioned tool indices: 5, 6, 9, and 11
	// Let's especially check those indices
	problematicIndices := []int{5, 6, 9, 11}

	fmt.Printf("Total number of registered tools: %d\n", len(tools))

	// If we have enough tools, check the problematic indices specifically
	if len(tools) > 11 {
		fmt.Println("Checking specifically mentioned problematic tool indices:")
		for _, idx := range problematicIndices {
			if idx < len(tools) {
				tool := tools[idx]
				fmt.Printf("Tool at index %d: %s\n", idx, tool.Name)
				validateToolSchema(t, tool)
			}
		}
	}

	// Test all tools
	for i, tool := range tools {
		t.Run(fmt.Sprintf("Tool_%d_%s", i, tool.Name), func(t *testing.T) {
			validateToolSchema(t, tool)
		})
	}
}

// validateToolSchema validates that a tool's schema complies with MCP requirements
func validateToolSchema(t *testing.T, tool mcp.Tool) {
	// Test basic tool properties
	assert.NotEmpty(t, tool.Name, "Tool name should not be empty")
	assert.NotEmpty(t, tool.Description, "Tool description should not be empty")

	// Convert the input schema to JSON for inspection
	schemaBytes, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err, "Failed to marshal input schema to JSON")

	// Parse back the schema
	var schema map[string]interface{}
	err = json.Unmarshal(schemaBytes, &schema)
	require.NoError(t, err, "Failed to unmarshal input schema from JSON")

	// Check for proper schema type and structure
	assert.Equal(t, "object", schema["type"], "Schema should have type 'object'")

	// Check properties if they exist
	properties, hasProps := schema["properties"].(map[string]interface{})
	if !hasProps {
		// Some tools might not have properties, which is valid
		return
	}

	// Check each property
	for propName, propValue := range properties {
		propMap, ok := propValue.(map[string]interface{})
		require.True(t, ok, "Property %s should be a map", propName)

		// If property is an array, it must have 'items' defined
		propType, hasType := propMap["type"]
		if hasType && propType == "array" {
			// The key validation: array must have items
			items, hasItems := propMap["items"]
			assert.True(t, hasItems, "Array property '%s' must have 'items' defined", propName)
			assert.NotNil(t, items, "Array property '%s' items must not be null", propName)

			// Items must be an object
			itemsObj, isObj := items.(map[string]interface{})
			assert.True(t, isObj, "Array property '%s' items must be an object", propName)

			if isObj {
				// Items must have a type
				itemType, hasItemType := itemsObj["type"]
				assert.True(t, hasItemType, "Array property '%s' items must have 'type' defined", propName)
				assert.NotEmpty(t, itemType, "Array property '%s' items type must not be empty", propName)
			}
		}
	}
}

// TestArrayItemsNotNull specifically tests that items for array parameters are not null
// This targets the specific AWS error: "failed to satisfy constraint: Member must not be null"
func TestArrayItemsNotNull(t *testing.T) {
	// Create a new server instance
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	err := s.Initialize(srv)
	require.NoError(t, err, "Server initialization should not fail")

	// Get all registered tools
	tools := srv.GetTools()

	// Map of tools known to have array parameters
	toolsWithArrayParams := map[string][]string{
		"check_docker_tags":    {"filterTags"},
		"check_python_versions": {"requirements"},
		"check_maven_versions":  {"dependencies"},
		"check_gradle_versions": {"dependencies"},
		"check_swift_versions":  {"dependencies"},
		"check_github_actions":  {"actions"},
	}

	// Find and test each tool with array parameters
	for toolName, arrayParams := range toolsWithArrayParams {
		var foundTool *mcp.Tool
		for i, tool := range tools {
			if tool.Name == toolName {
				foundTool = &tools[i]
				break
			}
		}

		if foundTool == nil {
			t.Errorf("Tool %s not found", toolName)
			continue
		}

		t.Run(toolName, func(t *testing.T) {
			// Convert the input schema to JSON for inspection
			schemaBytes, err := json.Marshal(foundTool.InputSchema)
			require.NoError(t, err, "Failed to marshal input schema to JSON")

			// Parse back the schema
			var schema map[string]interface{}
			err = json.Unmarshal(schemaBytes, &schema)
			require.NoError(t, err, "Failed to unmarshal input schema from JSON")

			// Check properties
			properties, hasProps := schema["properties"].(map[string]interface{})
			require.True(t, hasProps, "Schema should have properties")

			// Check each expected array parameter
			for _, paramName := range arrayParams {
				paramValue, hasProp := properties[paramName]
				require.True(t, hasProp, "Schema should have property '%s'", paramName)

				paramObj, isObj := paramValue.(map[string]interface{})
				require.True(t, isObj, "Property '%s' should be an object", paramName)

				// Verify it's an array
				paramType, hasType := paramObj["type"]
				require.True(t, hasType, "Property '%s' should have type", paramName)
				assert.Equal(t, "array", paramType, "Property '%s' should be of type array", paramName)

				// Verify it has items properly defined
				items, hasItems := paramObj["items"]
				assert.True(t, hasItems, "Array property '%s' must have 'items' defined", paramName)
				assert.NotNil(t, items, "Array property '%s' items must not be null", paramName)

				// Items must be an object with a type
				itemsObj, isObj := items.(map[string]interface{})
				assert.True(t, isObj, "Array property '%s' items must be an object", paramName)

				if isObj {
					itemType, hasItemType := itemsObj["type"]
					assert.True(t, hasItemType, "Array property '%s' items must have 'type' defined", paramName)
					assert.NotEmpty(t, itemType, "Array property '%s' items type must not be empty", paramName)
				}
			}
		})
	}
}

// TestGeneratedSchemaJSON tests that the generated schema JSON for tools
// has the correct structure, particularly for array parameters
func TestGeneratedSchemaJSON(t *testing.T) {
	// Create a new server instance
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	s.Initialize(srv)

	// Get the Docker tool specifically since it was mentioned as problematic
	var dockerTool *mcp.Tool
	for _, tool := range srv.GetTools() {
		if tool.Name == "check_docker_tags" {
			dockerTool = &tool
			break
		}
	}

	require.NotNil(t, dockerTool, "Docker tool should be registered")

	// Marshal the tool's schema to JSON
	schemaJSON, err := json.MarshalIndent(dockerTool.InputSchema, "", "  ")
	require.NoError(t, err, "Failed to marshal schema to JSON")

	// Verify the generated JSON contains the items property for filterTags
	jsonString := string(schemaJSON)
	assert.Contains(t, jsonString, `"filterTags"`)
	assert.Contains(t, jsonString, `"items"`)
	assert.Contains(t, jsonString, `"type": "string"`)

	// Unmarshal back to verify structure
	var schema map[string]interface{}
	err = json.Unmarshal(schemaJSON, &schema)
	require.NoError(t, err, "Failed to unmarshal schema JSON")

	// Navigate to the filterTags property
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "Schema should have properties")

	filterTags, ok := properties["filterTags"].(map[string]interface{})
	require.True(t, ok, "Schema should have filterTags property")

	items, ok := filterTags["items"].(map[string]interface{})
	require.True(t, ok, "filterTags should have items object")

	itemType, ok := items["type"].(string)
	require.True(t, ok, "items should have type")
	assert.Equal(t, "string", itemType, "filterTags items should be of type string")
}
