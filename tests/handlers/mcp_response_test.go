package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPToolResponse validates that tool responses adhere to the MCP protocol specification
func TestMCPToolResponse(t *testing.T) {
	// Skip test in CI since it makes remote API calls
	t.Skip("Skipping tests that make remote API calls")

	// Define handlers to test (commented out for now)
	/*
	// Create a logger for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a shared cache for testing
	sharedCache := &sync.Map{}

	// Define test cases for different handlers
	testCases := []struct {
		name      string
		handler   interface{}
		args      map[string]interface{}
		assertFn  func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "DockerHandler",
			handler: handlers.NewDockerHandler(logger, sharedCache),
			args: map[string]interface{}{
				"image":    "debian",
				"registry": "dockerhub",
				"limit":    float64(2),
			},
			assertFn: func(t *testing.T, result *mcp.CallToolResult) {
				validateDockerResult(t, result)
			},
		},
		{
			name: "PythonHandler",
			handler: handlers.NewPythonHandler(logger, sharedCache),
			args: map[string]interface{}{
				"requirements": []interface{}{"requests==2.25.1", "flask>=2.0.0"},
			},
			assertFn: func(t *testing.T, result *mcp.CallToolResult) {
				validatePythonResult(t, result)
			},
		},
		{
			name: "NPMHandler",
			handler: handlers.NewNpmHandler(logger, sharedCache),
			args: map[string]interface{}{
				"dependencies": map[string]interface{}{
					"express": "^4.17.1",
					"lodash":  "^4.17.21",
				},
			},
			assertFn: func(t *testing.T, result *mcp.CallToolResult) {
				validateNPMResult(t, result)
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the appropriate method based on handler type
			var result *mcp.CallToolResult
			var err error

			switch h := tc.handler.(type) {
			case *handlers.DockerHandler:
				result, err = h.GetLatestVersion(context.Background(), tc.args)
			case *handlers.PythonHandler:
				result, err = h.GetLatestVersionFromRequirements(context.Background(), tc.args)
			case *handlers.NpmHandler:
				result, err = h.GetLatestVersion(context.Background(), tc.args)
			default:
				t.Fatalf("Unknown handler type: %T", tc.handler)
			}

			// Validate the result
			require.NoError(t, err)
			require.NotNil(t, result, "Result should not be nil")

			// Validate that the result adheres to MCP protocol standards
			validateMCPToolResult(t, result)

			// Run handler-specific validations
			tc.assertFn(t, result)
		})
	}
	*/
}

// validateMCPToolResult validates that a tool response adheres to MCP protocol standards
func validateMCPToolResult(t *testing.T, result *mcp.CallToolResult) {
	// Check basic structure
	require.NotNil(t, result, "Tool result should not be nil")
	require.NotNil(t, result.Content, "Tool result content should not be nil")
	require.Greater(t, len(result.Content), 0, "Tool result should have content")

	// All MCP tool responses should have at least one content item
	// The first content item should be text for our JSON responses
	textContent, ok := result.Content[0].(*mcp.TextContent)
	assert.True(t, ok, "First content item should be text")

	if ok && textContent != nil {
		assert.NotEmpty(t, textContent.Text, "Text content should not be empty")
	}
}

// validateDockerResult validates Docker-specific response format
func validateDockerResult(t *testing.T, result *mcp.CallToolResult) {
	// First, validate it's a valid JSON array
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Docker result first content should be text")

	// Parse the JSON
	var dockerTags []handlers.DockerImageVersion
	err := json.Unmarshal([]byte(textContent.Text), &dockerTags)
	assert.NoError(t, err, "Docker result should be a valid JSON array of DockerImageVersion")

	// Validate the structure of at least one tag
	if len(dockerTags) > 0 {
		assert.NotEmpty(t, dockerTags[0].Name, "Docker tag should have a name")
		assert.NotEmpty(t, dockerTags[0].Tag, "Docker tag should have a tag")
		assert.NotEmpty(t, dockerTags[0].Registry, "Docker tag should have a registry")
	}
}

// validatePythonResult validates Python-specific response format
func validatePythonResult(t *testing.T, result *mcp.CallToolResult) {
	// First, validate it's a valid JSON array
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Python result first content should be text")

	// Parse the JSON
	var pythonPackages []handlers.PackageVersion
	err := json.Unmarshal([]byte(textContent.Text), &pythonPackages)
	assert.NoError(t, err, "Python result should be a valid JSON array of PackageVersion")

	// Validate the structure of at least one package
	if len(pythonPackages) > 0 {
		assert.NotEmpty(t, pythonPackages[0].Name, "Python package should have a name")
		assert.NotEmpty(t, pythonPackages[0].LatestVersion, "Python package should have a latest version")
	}
}

// validateNPMResult validates NPM-specific response format
func validateNPMResult(t *testing.T, result *mcp.CallToolResult) {
	// First, validate it's a valid JSON array
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "NPM result first content should be text")

	// Parse the JSON
	var npmPackages []handlers.PackageVersion
	err := json.Unmarshal([]byte(textContent.Text), &npmPackages)
	assert.NoError(t, err, "NPM result should be a valid JSON array of PackageVersion")

	// Validate the structure of at least one package
	if len(npmPackages) > 0 {
		assert.NotEmpty(t, npmPackages[0].Name, "NPM package should have a name")
		assert.NotEmpty(t, npmPackages[0].LatestVersion, "NPM package should have a latest version")
	}
}
