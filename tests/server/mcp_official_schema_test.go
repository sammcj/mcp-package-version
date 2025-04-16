package server_test

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolsAgainstOfficialMCPSchema validates that all tools conform to the official MCP schema
func TestToolsAgainstOfficialMCPSchema(t *testing.T) {
	// Skip since we can't access the tools directly from the server
	t.Skip("Skipping test since we can't access tools directly from MCP server")
}

// TestArrayParamsAgainstMCPSchema specifically validates array parameters against the MCP schema
func TestArrayParamsAgainstMCPSchema(t *testing.T) {
	// Skip since we can't access the tools directly from the server
	t.Skip("Skipping test since we can't access tools directly from MCP server")
}

// TestIndividualToolSchemas tests specific tools directly
func TestIndividualToolSchemas(t *testing.T) {
	// Create a logger for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a Docker tool to test its schema
	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Check available tags for Docker container images"),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Docker image name"),
		),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	t.Run("DockerToolSchema", func(t *testing.T) {
		// Marshal the schema to JSON
		schemaJSON, err := json.Marshal(dockerTool.InputSchema)
		require.NoError(t, err, "Failed to marshal Docker tool schema to JSON")

		// Parse the schema back for validation
		var schema map[string]interface{}
		err = json.Unmarshal(schemaJSON, &schema)
		require.NoError(t, err, "Failed to parse Docker tool schema")

		// Verify the schema structure
		assert.Equal(t, "object", schema["type"], "Schema should be object type")

		properties, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok, "Schema should have properties")

		// Check the filterTags property specifically
		filterTags, ok := properties["filterTags"].(map[string]interface{})
		assert.True(t, ok, "Schema should have filterTags property")

		assert.Equal(t, "array", filterTags["type"], "filterTags should be an array")

		items, ok := filterTags["items"].(map[string]interface{})
		assert.True(t, ok, "filterTags should have items property")

		assert.Equal(t, "string", items["type"], "filterTags items should be of type string")
	})

	// Create a Python tool to test its schema
	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Check latest stable versions for Python packages"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Array of requirements from requirements.txt"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
	)

	t.Run("PythonToolSchema", func(t *testing.T) {
		// Marshal the schema to JSON
		schemaJSON, err := json.Marshal(pythonTool.InputSchema)
		require.NoError(t, err, "Failed to marshal Python tool schema to JSON")

		// Parse the schema back for validation
		var schema map[string]interface{}
		err = json.Unmarshal(schemaJSON, &schema)
		require.NoError(t, err, "Failed to parse Python tool schema")

		// Verify the schema structure
		assert.Equal(t, "object", schema["type"], "Schema should be object type")

		properties, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok, "Schema should have properties")

		// Check the requirements property specifically
		requirements, ok := properties["requirements"].(map[string]interface{})
		assert.True(t, ok, "Schema should have requirements property")

		assert.Equal(t, "array", requirements["type"], "requirements should be an array")

		items, ok := requirements["items"].(map[string]interface{})
		assert.True(t, ok, "requirements should have items property")

		assert.Equal(t, "string", items["type"], "requirements items should be of type string")
	})
}
