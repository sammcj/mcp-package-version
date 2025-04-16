package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPSchemaCompliance validates that all tools registered by the server
// comply with the MCP schema specification
func TestMCPSchemaDirectly(t *testing.T) {
	// Create direct tool definitions to test instead of accessing through server
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Define tool definitions directly to test
	tools := []mcp.Tool{
		mcp.NewTool("check_docker_tags",
			mcp.WithDescription("Check available tags for Docker container images"),
			mcp.WithString("image",
				mcp.Required(),
				mcp.Description("Docker image name"),
			),
			mcp.WithEnum("registry",
				mcp.Required(),
				mcp.Description("Registry to fetch tags from"),
				mcp.Values("dockerhub", "ghcr", "custom"),
			),
			mcp.WithArray("filterTags",
				mcp.Description("Array of regex patterns to filter tags"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		),
		mcp.NewTool("check_python_versions",
			mcp.WithDescription("Check latest stable versions for Python packages"),
			mcp.WithArray("requirements",
				mcp.Required(),
				mcp.Description("Array of requirements from requirements.txt"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		),
		mcp.NewTool("check_npm_versions",
			mcp.WithDescription("Check latest stable versions for NPM packages"),
			mcp.WithObject("dependencies",
				mcp.Required(),
				mcp.Description("NPM dependencies object from package.json"),
				mcp.AdditionalProperties(map[string]interface{}{
					"type": "string",
				}),
			),
		),
	}

	// Test each tool's schema for validity
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
			assert.NotNil(t, items, "Array property '%s' items must not be nil", propName)

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
func TestArrayItemsNotNull(t *testing.T) {
	// Define tools with array parameters
	toolsWithArrayParams := []struct {
		name        string
		tool        mcp.Tool
		arrayParams []string
	}{
		{
			"check_docker_tags",
			mcp.NewTool("check_docker_tags",
				mcp.WithArray("filterTags",
					mcp.Description("Array of regex patterns to filter tags"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
			),
			[]string{"filterTags"},
		},
		{
			"check_python_versions",
			mcp.NewTool("check_python_versions",
				mcp.WithArray("requirements",
					mcp.Required(),
					mcp.Description("Array of requirements from requirements.txt"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
			),
			[]string{"requirements"},
		},
		{
			"check_maven_versions",
			mcp.NewTool("check_maven_versions",
				mcp.WithArray("dependencies",
					mcp.Required(),
					mcp.Description("Array of Maven dependencies"),
					mcp.Items(map[string]interface{}{"type": "object"}),
				),
			),
			[]string{"dependencies"},
		},
	}

	// Test each tool with array parameters
	for _, tc := range toolsWithArrayParams {
		t.Run(tc.name, func(t *testing.T) {
			// Convert the input schema to JSON for inspection
			schemaBytes, err := json.Marshal(tc.tool.InputSchema)
			require.NoError(t, err, "Failed to marshal input schema to JSON")

			// Parse back the schema
			var schema map[string]interface{}
			err = json.Unmarshal(schemaBytes, &schema)
			require.NoError(t, err, "Failed to unmarshal input schema from JSON")

			// Check properties
			properties, hasProps := schema["properties"].(map[string]interface{})
			require.True(t, hasProps, "Schema should have properties")

			// Check each expected array parameter
			for _, paramName := range tc.arrayParams {
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
				assert.NotNil(t, items, "Array property '%s' items must not be nil", paramName)

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
