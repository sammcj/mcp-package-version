package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sammcj/mcp-package-version/v2/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

// TestToolsAgainstOfficialMCPSchema validates that all tools conform to the official MCP schema
func TestToolsAgainstOfficialMCPSchema(t *testing.T) {
	// Skip if we can't access the official schema
	resp, err := http.Get("https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/refs/heads/main/schema/2025-03-26/schema.json")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skip("Could not access official MCP schema, skipping test")
	}
	defer resp.Body.Close()

	// Read the official schema
	schemaData, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read schema data")

	// Parse the schema
	schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
	schema, err := gojsonschema.NewSchema(schemaLoader)
	require.NoError(t, err, "Failed to parse official MCP schema")

	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	err = s.Initialize(srv)
	require.NoError(t, err, "Server initialization should not fail")

	// Get all registered tools
	tools := srv.GetTools()
	require.NotEmpty(t, tools, "Server should have registered tools")

	// Test each tool's schema for validity against the official MCP schema
	for _, tool := range tools {
		t.Run("Tool_"+tool.Name, func(t *testing.T) {
			// Create a tool definition conforming to the MCP schema format
			toolDef := map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			}

			// Convert to JSON for validation
			toolJSON, err := json.Marshal(toolDef)
			require.NoError(t, err, "Failed to marshal tool definition")

			// Validate against the Tool part of the MCP schema
			// Note: This section might need adjustment based on the exact structure of the MCP schema
			documentLoader := gojsonschema.NewStringLoader(string(toolJSON))
			result, err := schema.Validate(documentLoader)
			require.NoError(t, err, "Schema validation failed")

			// Check validation result
			if !result.Valid() {
				for _, desc := range result.Errors() {
					t.Errorf("Schema validation error: %s", desc)
				}
				t.Fail()
			}
		})
	}
}

// TestArrayParamsAgainstMCPSchema specifically validates array parameters against the MCP schema
func TestArrayParamsAgainstMCPSchema(t *testing.T) {
	// Create a new server instance
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	err := s.Initialize(srv)
	require.NoError(t, err, "Server initialization should not fail")

	// Map of tools known to have array parameters
	toolsWithArrayParams := map[string][]string{
		"check_docker_tags":    {"filterTags"},
		"check_python_versions": {"requirements"},
		"check_maven_versions":  {"dependencies"},
		"check_gradle_versions": {"dependencies"},
		"check_swift_versions":  {"dependencies"},
		"check_github_actions":  {"actions"},
	}

	// Find each tool with array parameters
	for toolName, arrayParams := range toolsWithArrayParams {
		var foundTool *mcp.Tool
		for i, tool := range srv.GetTools() {
			if tool.Name == toolName {
				foundTool = &srv.GetTools()[i]
				break
			}
		}
		require.NotNil(t, foundTool, "Tool %s should be registered", toolName)

		t.Run(toolName, func(t *testing.T) {
			// Convert schema to JSON for analysis
			schemaJSON, err := json.Marshal(foundTool.InputSchema)
			require.NoError(t, err, "Failed to marshal schema to JSON")

			// Check schema structure
			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			require.NoError(t, err, "Failed to unmarshal schema")

			// Ensure it's a valid JSON Schema
			require.Equal(t, "object", schema["type"], "Schema should be an object type")

			properties, ok := schema["properties"].(map[string]interface{})
			require.True(t, ok, "Schema should have properties")

			// Test each array parameter
			for _, paramName := range arrayParams {
				paramSchema, ok := properties[paramName].(map[string]interface{})
				require.True(t, ok, "Schema should have property %s", paramName)
				require.Equal(t, "array", paramSchema["type"], "Property %s should be an array", paramName)

				// Key validation: array must have items defined and not null
				items, hasItems := paramSchema["items"]
				require.True(t, hasItems, "Array property %s must have items defined", paramName)
				require.NotNil(t, items, "Array property %s items must not be null", paramName)

				// Items must have a valid type
				itemsMap, ok := items.(map[string]interface{})
				require.True(t, ok, "Items for %s must be a valid object", paramName)

				itemType, hasType := itemsMap["type"]
				require.True(t, hasType, "Items for %s must have a type", paramName)
				require.NotEmpty(t, itemType, "Items type for %s must not be empty", paramName)
			}
		})
	}
}

// TestSpecificProblemTools tests the specific tools that were reported as problematic with AWS
func TestSpecificProblemTools(t *testing.T) {
	// Create a new server instance
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	s := server.NewPackageVersionServer("test", "test", "test")

	// Create a new MCP server
	srv := mcpserver.NewMCPServer("test-server", "Test Package Version Server")

	// Initialize the server, which registers all tools
	err := s.Initialize(srv)
	require.NoError(t, err, "Server initialization should not fail")

	// Get all tools
	tools := srv.GetTools()

	// The error mentioned tools at indices 5, 6, 9, and 11
	problematicIndices := []int{5, 6, 9, 11}

	// Check if we have enough tools to test
	if len(tools) <= 11 {
		t.Skipf("Not enough tools to test problematic indices (got %d, need at least 12)", len(tools))
		return
	}

	// Test each of the problematic tools
	for _, idx := range problematicIndices {
		tool := tools[idx]
		t.Run(tool.Name, func(t *testing.T) {
			// Convert schema to JSON for analysis
			schemaJSON, err := json.Marshal(tool.InputSchema)
			require.NoError(t, err, "Failed to marshal schema to JSON")

			// Validate schema structure
			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			require.NoError(t, err, "Failed to unmarshal schema")

			require.Equal(t, "object", schema["type"], "Schema should be an object type")

			// Check properties
			properties, hasProps := schema["properties"].(map[string]interface{})
			require.True(t, hasProps, "Schema should have properties")

			// Look for array properties
			for propName, propValue := range properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					continue
				}

				propType, hasType := propMap["type"]
				if !hasType || propType != "array" {
					continue
				}

				// This is an array property, validate its items
				items, hasItems := propMap["items"]
				require.True(t, hasItems, "Array property %s must have items defined", propName)
				require.NotNil(t, items, "Array property %s items must not be null", propName)

				itemsMap, ok := items.(map[string]interface{})
				require.True(t, ok, "Items for %s must be a valid object, got %T", propName, items)

				itemType, hasItemType := itemsMap["type"]
				require.True(t, hasItemType, "Items for %s must have a type", propName)
				require.NotEmpty(t, itemType, "Items type for %s must not be empty", propName)
			}
		})
	}
}
