// Package tests provides testing utilities for MCP handlers
package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// MockResponse represents a mocked HTTP response
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// MockClient is a mock HTTP client for testing
type MockClient struct {
	Responses map[string]MockResponse
}

// NewMockClient creates a new mock HTTP client
func NewMockClient() *MockClient {
	return &MockClient{
		Responses: make(map[string]MockResponse),
	}
}

// AddMockResponse adds a mock response for a specific URL pattern
func (c *MockClient) AddMockResponse(urlPattern string, response MockResponse) {
	c.Responses[urlPattern] = response
}

// Do implements the http.Client interface
func (c *MockClient) Do(req *http.Request) (*http.Response, error) {
	// Find matching response based on URL pattern
	var mockResp MockResponse
	found := false

	// Find the first matching URL pattern
	for pattern, resp := range c.Responses {
		if strings.Contains(req.URL.String(), pattern) {
			mockResp = resp
			found = true
			break
		}
	}

	// Default response if no match found
	if !found {
		mockResp = MockResponse{
			StatusCode: http.StatusNotFound,
			Body:       `{"error": "No mock response found for this URL"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}
	}

	// Create response headers
	headers := make(http.Header)
	for k, v := range mockResp.Headers {
		headers.Add(k, v)
	}

	// Create response
	response := &http.Response{
		StatusCode: mockResp.StatusCode,
		Body:       io.NopCloser(bytes.NewBufferString(mockResp.Body)),
		Header:     headers,
		Request:    req,
	}

	return response, nil
}

// Helper functions for creating common mock responses

// AddDockerHubTagsResponse adds a mock response for Docker Hub tags
func (c *MockClient) AddDockerHubTagsResponse(image string, tags []string) {
	type DockerHubTag struct {
		Name string `json:"name"`
	}

	type DockerHubResponse struct {
		Results []DockerHubTag `json:"results"`
	}

	// Create response with tags
	tagResults := make([]DockerHubTag, 0, len(tags))
	for _, tag := range tags {
		tagResults = append(tagResults, DockerHubTag{Name: tag})
	}

	response := DockerHubResponse{
		Results: tagResults,
	}

	// Convert to JSON
	respBody, _ := json.Marshal(response)

	c.AddMockResponse(
		"registry.hub.docker.com/v2/repositories/"+image+"/tags",
		MockResponse{
			StatusCode: 200,
			Body:       string(respBody),
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	)
}

// AddGHCRTagsResponse adds a mock response for GitHub Container Registry
func (c *MockClient) AddGHCRTagsResponse(image string, tags []string) {
	type GHCRTag struct {
		Name string `json:"name"`
	}

	// Create response with tags
	tagResults := make([]GHCRTag, 0, len(tags))
	for _, tag := range tags {
		tagResults = append(tagResults, GHCRTag{Name: tag})
	}

	// Convert to JSON
	respBody, _ := json.Marshal(tagResults)

	c.AddMockResponse(
		"ghcr.io/v2/"+image+"/tags/list",
		MockResponse{
			StatusCode: 200,
			Body:       string(respBody),
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	)
}

// AddNPMPackageResponse adds a mock response for NPM registry
func (c *MockClient) AddNPMPackageResponse(packageName string, versions map[string]interface{}) {
	type NPMResponse struct {
		Versions map[string]interface{} `json:"versions"`
	}

	response := NPMResponse{
		Versions: versions,
	}

	// Convert to JSON
	respBody, _ := json.Marshal(response)

	c.AddMockResponse(
		"registry.npmjs.org/"+packageName,
		MockResponse{
			StatusCode: 200,
			Body:       string(respBody),
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	)
}

// AddPyPIPackageResponse adds a mock response for PyPI registry
func (c *MockClient) AddPyPIPackageResponse(packageName string, releases map[string][]interface{}) {
	type PyPIResponse struct {
		Releases map[string][]interface{} `json:"releases"`
	}

	response := PyPIResponse{
		Releases: releases,
	}

	// Convert to JSON
	respBody, _ := json.Marshal(response)

	c.AddMockResponse(
		"pypi.org/pypi/"+packageName+"/json",
		MockResponse{
			StatusCode: 200,
			Body:       string(respBody),
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	)
}

// AddGoPackageResponse adds a mock response for Go package info
func (c *MockClient) AddGoPackageResponse(packageName string, versions []string) {
	// Mock proxy.golang.org response
	c.AddMockResponse(
		"proxy.golang.org/"+packageName+"/@v/list",
		MockResponse{
			StatusCode: 200,
			Body:       strings.Join(versions, "\n"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
		},
	)
}

// Add error response
func (c *MockClient) AddErrorResponse(urlPattern string, statusCode int, errorMessage string) {
	c.AddMockResponse(
		urlPattern,
		MockResponse{
			StatusCode: statusCode,
			Body:       `{"error": "` + errorMessage + `"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	)
}
