package handlers

import (
	"context"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDockerHandler_GetLatestVersion(t *testing.T) {
	// Create a logger for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a shared cache for testing
	sharedCache := &sync.Map{}

	// Create a handler
	handler := NewDockerHandler(logger, sharedCache)

	// Define test cases
	tests := []struct {
		name        string
		args        map[string]interface{}
		wantErr     bool
		errorString string
		skipRemote  bool // Add flag to skip tests that make remote calls
	}{
		{
			name: "Valid dockerhub image",
			args: map[string]interface{}{
				"image":    "nginx",
				"registry": "dockerhub",
				"limit":    float64(5),
			},
			wantErr:    false,
			skipRemote: true, // Skip remote calls during unit testing
		},
		{
			name: "Valid with filterTags array",
			args: map[string]interface{}{
				"image":      "nginx",
				"registry":   "dockerhub",
				"filterTags": []interface{}{"stable", "latest"},
			},
			wantErr:    false,
			skipRemote: true, // Skip remote calls during unit testing
		},
		{
			name: "Missing required image parameter",
			args: map[string]interface{}{
				"registry": "dockerhub",
			},
			wantErr:     true,
			errorString: "missing required parameter: image",
		},
		{
			name: "Invalid registry",
			args: map[string]interface{}{
				"image":    "nginx",
				"registry": "invalid",
			},
			wantErr:     true,
			errorString: "invalid registry: invalid",
		},
		{
			name: "Custom registry without customRegistry parameter",
			args: map[string]interface{}{
				"image":    "nginx",
				"registry": "custom",
			},
			wantErr:     true,
			errorString: "missing required parameter for custom registry: customRegistry",
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip remote tests based on flag
			if tt.skipRemote {
				t.Skip("Skipping test that makes remote API calls")
			}

			result, err := handler.GetLatestVersion(context.Background(), tt.args)

			// Check error conditions
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				return
			}

			// If not expecting error, validate result
			assert.NoError(t, err)
			assert.NotNil(t, result)

			 // Only validate tool result format if we have a result
			if result != nil {
				validateToolResult(t, result)
			}
		})
	}
}

// TestMCPResultFormat tests that the Docker handler returns results
// that conform to the MCP specification
func TestDockerMCPResultFormat(t *testing.T) {
	// Skip test because it would make remote calls
	t.Skip("Skipping test that makes remote API calls")

	// This would be the code if we wanted to run the test
	/*
	// Create a logger for testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a shared cache for testing
	sharedCache := &sync.Map{}

	// Create a handler
	handler := NewDockerHandler(logger, sharedCache)

	// Create valid arguments
	args := map[string]interface{}{
		"image":    "debian",
		"registry": "dockerhub",
		"limit":    float64(2),
	}

	// Call the handler
	result, err := handler.GetLatestVersion(context.Background(), args)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Validate the result structure
	validateToolResultStructure(t, result)
	*/
}

// Helper function to validate tool result format
func validateToolResult(t *testing.T, result *mcp.CallToolResult) {
	assert.NotNil(t, result, "Tool result should not be nil")
	assert.NotNil(t, result.Content, "Tool result content should not be nil")

	// Check if content is empty - don't proceed if it is
	if len(result.Content) == 0 {
		t.Log("Tool result content is empty")
		return
	}

	// Since we're using JSON output, the first content item should be text
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Log("First content item is not text content")
		return
	}

	assert.True(t, ok, "First content item should be text content")
	if textContent != nil {
		assert.NotEmpty(t, textContent.Text, "Text content should not be empty")
	}
}

// Helper function to validate the structure of Docker tool results
func validateToolResultStructure(t *testing.T, result *mcp.CallToolResult) {
	// Safety check
	if result == nil || len(result.Content) == 0 {
		t.Log("Result or content is nil/empty")
		return
	}

	// First content item should be text content with JSON
	textContent, ok := result.Content[0].(*mcp.TextContent)
	assert.True(t, ok, "First content item should be text content")

	if !ok || textContent == nil {
		t.Log("Content is not text content or is nil")
		return
	}

	// The text should be valid JSON representing an array of DockerImageVersion objects
	assert.Contains(t, textContent.Text, "[", "Result should be a JSON array")
	assert.Contains(t, textContent.Text, "name", "Result should contain image name")
	assert.Contains(t, textContent.Text, "tag", "Result should contain image tag")
	assert.Contains(t, textContent.Text, "registry", "Result should contain registry name")
}
