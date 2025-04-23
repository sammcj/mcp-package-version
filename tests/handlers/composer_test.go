package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sammcj/mcp-package-version/v2/internal/handlers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockHTTPClient mocks the HTTP client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestIsLaravelPackage(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		expected    bool
	}{
		{
			name:        "illuminate vendor package",
			packageName: "illuminate/support",
			expected:    true,
		},
		{
			name:        "spatie vendor package",
			packageName: "spatie/laravel-permission",
			expected:    true,
		},
		{
			name:        "filament vendor package",
			packageName: "filament/forms",
			expected:    true,
		},
		{
			name:        "laravel prefixed package",
			packageName: "vendor/laravel-package",
			expected:    true,
		},
		{
			name:        "filament prefixed package",
			packageName: "vendor/filament-plugin",
			expected:    true,
		},
		{
			name:        "non-laravel package",
			packageName: "vendor/some-package",
			expected:    false,
		},
		{
			name:        "invalid package format",
			packageName: "invalid-package-name",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handlers.IsLaravelPackage(tt.packageName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createMockResponse(name string) *handlers.PackagistResponse {
	resp := &handlers.PackagistResponse{}
	resp.Package.Name = name
	resp.Package.Description = "Laravel Package"
	resp.Package.Versions = make(map[string]struct {
		Version     string    `json:"version"`
		VersionNorm string    `json:"version_normalized"`
		Time        time.Time `json:"time"`
	})

	resp.Package.Versions["10.0.0"] = struct {
		Version     string    `json:"version"`
		VersionNorm string    `json:"version_normalized"`
		Time        time.Time `json:"time"`
	}{
		Version:     "10.0.0",
		VersionNorm: "10.0.0.0",
		Time:        time.Date(2023, 2, 14, 12, 0, 0, 0, time.UTC),
	}

	resp.Package.Versions["9.0.0"] = struct {
		Version     string    `json:"version"`
		VersionNorm string    `json:"version_normalized"`
		Time        time.Time `json:"time"`
	}{
		Version:     "9.0.0",
		VersionNorm: "9.0.0.0",
		Time:        time.Date(2022, 2, 8, 12, 0, 0, 0, time.UTC),
	}

	return resp
}

func TestComposerHandler_GetLatestVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Extract package name from URL using the correct /p2/ path
			url := req.URL.String()
			parts := strings.Split(url, "/p2/") // Use /p2/ instead of /packages/
			if len(parts) != 2 {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte("invalid URL"))),
				}, nil
			}
			packagePath := strings.TrimSuffix(parts[1], ".json")
			packageParts := strings.Split(packagePath, "/")
			if len(packageParts) != 2 {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte("invalid package format"))),
				}, nil
			}

			// Create appropriate mock response based on package
			mockResponse := createMockResponse(packagePath)
			responseJSON, _ := json.Marshal(mockResponse)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(responseJSON)),
			}, nil
		},
	}

	handler := handlers.NewComposerHandler(logger, nil)
	handler.SetClient(mockClient)

	args := map[string]interface{}{
		"dependencies": map[string]interface{}{
			"illuminate/support":         "^8.0",
			"spatie/laravel-permission":  "^5.0",
			"filament/forms":             "^2.0",
			"vendor/laravel-package":     "^1.0",
			"vendor/non-laravel-package": "^1.0",
			"invalid/format/extra/parts": "^1.0",
		},
		"constraints": map[string]interface{}{
			"spatie/laravel-permission": map[string]interface{}{
				"excludePackage": true,
			},
		},
	}

	result, err := handler.GetLatestVersion(context.Background(), args)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Validate the result content
	validateComposerToolResult(t, result)
}

// Helper function to validate composer tool result
func validateComposerToolResult(t *testing.T, result *mcp.CallToolResult) {
	assert.NotNil(t, result.Content, "Tool result content should not be nil")
	assert.Greater(t, len(result.Content), 0, "Tool result content should not be empty")

	// Get the text content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	assert.True(t, ok, "First content item should be text content")
	if !ok || textContent == nil {
		t.Log("Content is not text content or is nil")
		return
	}

	// Parse and validate specific test cases
	var versions []handlers.PackageVersion
	err := json.Unmarshal([]byte(textContent.Text), &versions)
	assert.NoError(t, err, "Should be able to unmarshal JSON result")

	// Test the number of results (should only include Laravel packages)
	laravelPackages := 0
	for _, version := range versions {
		if !version.Skipped {
			laravelPackages++
		}
	}
	assert.Greater(t, laravelPackages, 0, "Should have found some Laravel packages")

	// Test excluded package
	for _, version := range versions {
		if version.Name == "spatie/laravel-permission" {
			assert.True(t, version.Skipped, "Package should be skipped due to constraints")
			assert.Equal(t, "Package excluded by constraints", version.SkipReason)
		}
	}

	// Test invalid package format
	for _, version := range versions {
		if version.Name == "invalid/format/extra/parts" {
			assert.True(t, version.Skipped, "Invalid package format should be skipped")
			assert.Equal(t, "Invalid package name format", version.SkipReason)
		}
	}

	// Test non-Laravel package (should not be in results)
	found := false
	for _, result := range versions {
		if result.Name == "vendor/non-laravel-package" {
			found = true
			break
		}
	}
	assert.False(t, found, "Non-Laravel package should not be in results")
}
