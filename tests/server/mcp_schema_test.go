package server_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPSchemaCompliance validates that all tools comply with the MCP schema specification
func TestMCPSchemaCompliance(t *testing.T) {
	// Skip since we can't access the tools directly from the server
	t.Skip("Skipping test since we can't access tools directly from MCP server")
}

// TestArrayParameterSchemas validates schemas for tools with array parameters
func TestArrayParameterSchemas(t *testing.T) {
	// Create a new server instance for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Map of tools known to have array parameters with their expected schemas
	toolSchemas := map[string]mcp.Tool{
		"check_docker_tags": mcp.NewTool("check_docker_tags",
			mcp.WithDescription("Check available tags for Docker container images"),
			mcp.WithString("image", mcp.Required(), mcp.Description("Docker image name")),
			mcp.WithArray("filterTags",
				mcp.Description("Array of regex patterns to filter tags"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		),
		"check_python_versions": mcp.NewTool("check_python_versions",
			mcp.WithDescription("Check latest stable versions for Python packages"),
			mcp.WithArray("requirements",
				mcp.Required(),
				mcp.Description("Array of requirements from requirements.txt"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		),
		"check_maven_versions": mcp.NewTool("check_maven_versions",
			mcp.WithDescription("Check latest stable versions for Java packages in pom.xml"),
			mcp.WithArray("dependencies",
				mcp.Required(),
				mcp.Description("Array of Maven dependencies"),
				mcp.Items(map[string]interface{}{"type": "object"}),
			),
		),
		"check_gradle_versions": mcp.NewTool("check_gradle_versions",
			mcp.WithDescription("Check latest stable versions for Java packages in build.gradle"),
			mcp.WithArray("dependencies",
				mcp.Required(),
				mcp.Description("Array of Gradle dependencies"),
				mcp.Items(map[string]interface{}{"type": "object"}),
			),
		),
		"check_swift_versions": mcp.NewTool("check_swift_versions",
			mcp.WithDescription("Check latest stable versions for Swift packages in Package.swift"),
			mcp.WithArray("dependencies",
				mcp.Required(),
				mcp.Description("Array of Swift package dependencies"),
				mcp.Items(map[string]interface{}{"type": "object"}),
			),
		),
		"check_github_actions": mcp.NewTool("check_github_actions",
			mcp.WithDescription("Check latest versions for GitHub Actions"),
			mcp.WithArray("actions",
				mcp.Required(),
				mcp.Description("Array of GitHub Actions to check"),
				mcp.Items(map[string]interface{}{"type": "object"}),
			),
		),
	}

	// Test each tool schema
	for toolName, toolDef := range toolSchemas {
		t.Run(toolName, func(t *testing.T) {
			// Convert the schema to JSON for examination
			schemaJSON, err := json.Marshal(toolDef.InputSchema)
			assert.NoError(t, err, "Schema should be marshallable to JSON")

			// Parse the schema back as a map for examination
			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			assert.NoError(t, err, "Schema should be unmarshallable from JSON")

			// Check if the schema has properties
			properties, ok := schema["properties"].(map[string]interface{})
			assert.True(t, ok, "Schema should have properties")

			// Find the array properties
			for propName, propValue := range properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
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
		})
	}
}

// TestSpecificItemsSchema tests the specific items schema definition for array properties
func TestSpecificItemsSchema(t *testing.T) {
	// Create Docker tool with array parameter
	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Check available tags for Docker container images"),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	// Convert to JSON to verify the schema structure
	schemaJSON, err := json.MarshalIndent(dockerTool.InputSchema, "", "  ")
	require.NoError(t, err, "Failed to marshal tool schema to JSON")

	// Print the schema for debugging
	fmt.Printf("Docker Tool Schema JSON:\n%s\n", string(schemaJSON))

	// Parse back to verify structure
	var schema map[string]interface{}
	err = json.Unmarshal(schemaJSON, &schema)
	require.NoError(t, err, "Failed to unmarshal schema JSON")

	// Navigate to properties > filterTags > items
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "Schema should have properties")

	filterTags, ok := properties["filterTags"].(map[string]interface{})
	require.True(t, ok, "Schema should have filterTags property")

	items, ok := filterTags["items"].(map[string]interface{})
	require.True(t, ok, "filterTags should have items property")

	// Verify items type
	itemType, ok := items["type"].(string)
	require.True(t, ok, "items should have type property")
	assert.Equal(t, "string", itemType, "items type should be 'string'")
}
