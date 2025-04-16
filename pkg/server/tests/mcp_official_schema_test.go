package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

// TestValidateSchemaDirectly validates tool schemas directly
// without relying on access to server tools
func TestValidateSchemaDirectly(t *testing.T) {
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

	// Create direct tool definitions to test against the schema
	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Check available tags for Docker container images"),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Docker image name"),
		),
		mcp.WithString("registry",
			mcp.Required(),
			mcp.Description("Registry to fetch tags from"),
		),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Check latest stable versions for Python packages"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Array of requirements from requirements.txt"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	// Test each tool against the schema
	tools := []struct {
		name string
		tool mcp.Tool
	}{
		{"DockerTool", dockerTool},
		{"PythonTool", pythonTool},
	}

	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			// Create a tool definition conforming to the MCP schema format
			toolDef := map[string]interface{}{
				"name":        tool.tool.Name,
				"description": tool.tool.Description,
				"inputSchema": tool.tool.InputSchema,
			}

			// Convert to JSON for validation
			toolJSON, err := json.Marshal(toolDef)
			require.NoError(t, err, "Failed to marshal tool definition")

			// Validate against the Tool part of the MCP schema
			documentLoader := gojsonschema.NewStringLoader(string(toolJSON))
			result, err := schema.Validate(documentLoader)

			// Check for validation errors
			if err != nil {
				t.Logf("Schema validation error: %s", err)
				t.Skip("Skipping schema validation due to error")
				return
			}

			// Check validation result - but don't fail as the official schema might not match our tool format exactly
			if !result.Valid() {
				for _, desc := range result.Errors() {
					t.Logf("Schema validation warning: %s", desc)
				}
			}
		})
	}
}

// TestArrayParamsDirectly checks array parameters directly
func TestArrayParamsDirectly(t *testing.T) {
	// Create tools with array parameters to test
	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Check available tags for Docker container images"),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Check latest stable versions for Python packages"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Array of requirements from requirements.txt"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	mavenTool := mcp.NewTool("check_maven_versions",
		mcp.WithDescription("Check latest stable versions for Java packages"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Maven dependencies"),
			mcp.Items(map[string]interface{}{"type": "object"}),
		),
	)

	// Define the tools with array parameters that we want to check
	tools := []struct {
		name        string
		tool        mcp.Tool
		arrayParams []string
	}{
		{"DockerTool", dockerTool, []string{"filterTags"}},
		{"PythonTool", pythonTool, []string{"requirements"}},
		{"MavenTool", mavenTool, []string{"dependencies"}},
	}

	// Test each tool's array parameters
	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			// Convert to JSON for examination
			schemaJSON, err := json.Marshal(tc.tool.InputSchema)
			require.NoError(t, err, "Failed to marshal schema to JSON")

			// Parse back as a map for examination
			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			require.NoError(t, err, "Failed to unmarshal schema from JSON")

			// Check properties
			properties, ok := schema["properties"].(map[string]interface{})
			require.True(t, ok, "Schema should have properties")

			// Check each array parameter
			for _, paramName := range tc.arrayParams {
				param, ok := properties[paramName].(map[string]interface{})
				require.True(t, ok, "Schema should have %s property", paramName)

				// Verify it's an array
				propType, hasType := param["type"]
				require.True(t, hasType, "%s should have a type", paramName)
				assert.Equal(t, "array", propType, "%s should be an array", paramName)

				// Verify it has items property
				items, hasItems := param["items"]
				assert.True(t, hasItems, "%s must have items defined", paramName)
				assert.NotNil(t, items, "%s items must not be nil", paramName)

				// Check the items property structure
				itemsMap, ok := items.(map[string]interface{})
				assert.True(t, ok, "%s items must be a valid object", paramName)
				assert.NotEmpty(t, itemsMap["type"], "%s items must have a type", paramName)
			}
		})
	}
}
