package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

// ComposerHandler handles Composer/Laravel package version checking
type ComposerHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewComposerHandler creates a new Composer handler
func NewComposerHandler(logger *logrus.Logger, cache *sync.Map) *ComposerHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &ComposerHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// SetClient sets the HTTP client for testing
func (h *ComposerHandler) SetClient(client HTTPClient) {
	h.client = client
}

const (
	packagistAPIURL = "https://repo.packagist.org/packages/%s/%s.json"
)

var (
	laravelVendors = map[string]bool{
		"illuminate": true,
		"spatie":     true,
		"filament":   true,
	}
)

// GetLatestVersion gets information about Composer package versions
func (h *ComposerHandler) GetLatestVersion(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	h.logger.Debug("Getting Composer package version information")

	// Parse dependencies
	dependenciesRaw, ok := args["dependencies"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing required parameter: dependencies")
	}

	// Convert dependencies to string map
	dependencies := make(map[string]string)
	for pkg, ver := range dependenciesRaw {
		if verStr, ok := ver.(string); ok {
			dependencies[pkg] = verStr
		}
	}

	// Parse constraints
	var constraints VersionConstraints
	if constraintsRaw, ok := args["constraints"].(map[string]interface{}); ok {
		constraints = make(VersionConstraints)
		for pkg, constraint := range constraintsRaw {
			if c, ok := constraint.(map[string]interface{}); ok {
				var vc VersionConstraint
				if exclude, ok := c["excludePackage"].(bool); ok {
					vc.ExcludePackage = exclude
				}
				if major, ok := c["majorVersion"].(float64); ok {
					majorInt := int(major)
					vc.MajorVersion = &majorInt
				}
				constraints[pkg] = vc
			}
		}
	}

	// Get package versions
	results := h.checkComposerVersions(dependencies, constraints)

	// Convert results to JSON
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %v", err)
	}

	// Create text content
	textContent := &mcp.TextContent{
		Type: "text",
		Text: string(jsonBytes),
	}

	// Return result with text content
	return &mcp.CallToolResult{
		Content: []mcp.Content{textContent},
	}, nil
}

// IsLaravelPackage checks if a package is a Laravel package based on prefix or vendor
func IsLaravelPackage(packageName string) bool {
	parts := strings.Split(packageName, "/")
	if len(parts) != 2 {
		return false
	}

	vendor := parts[0]
	name := parts[1]

	// Check if it's from a known Laravel vendor
	if laravelVendors[vendor] {
		return true
	}

	// Check for Laravel-specific prefixes
	return strings.HasPrefix(name, "laravel-") || strings.HasPrefix(name, "filament-")
}

// PackagistResponse represents a response from the Packagist API
type PackagistResponse struct {
	Package struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Versions    map[string]struct {
			Version     string    `json:"version"`
			VersionNorm string    `json:"version_normalized"`
			Time        time.Time `json:"time"`
		} `json:"versions"`
	} `json:"package"`
}

// getLatestVersion retrieves the latest version of a package from Packagist
func (h *ComposerHandler) getLatestVersion(vendor, package_ string) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("packagist:%s/%s", vendor, package_)
	if cachedVersion, ok := h.cache.Load(cacheKey); ok {
		h.logger.WithField("package", fmt.Sprintf("%s/%s", vendor, package_)).Debug("Using cached version")
		return cachedVersion.(string), nil
	}

	url := fmt.Sprintf(packagistAPIURL, vendor, package_)
	h.logger.WithFields(logrus.Fields{
		"package": fmt.Sprintf("%s/%s", vendor, package_),
		"url":     url,
	}).Debug("Fetching Packagist package info")

	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %v", err)
	}

	var packageInfo PackagistResponse
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse package info: %v", err)
	}

	var latestVersion string
	var latestTime time.Time

	for _, v := range packageInfo.Package.Versions {
		// Skip dev versions
		if strings.Contains(v.Version, "dev-") || strings.Contains(v.Version, "-dev") {
			continue
		}

		if v.Time.After(latestTime) {
			latestTime = v.Time
			latestVersion = v.Version
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no stable versions found for %s/%s", vendor, package_)
	}

	// Cache the result
	h.cache.Store(cacheKey, latestVersion)

	return latestVersion, nil
}

// checkComposerVersions checks the latest versions for Laravel/Composer packages
func (h *ComposerHandler) checkComposerVersions(dependencies map[string]string, constraints VersionConstraints) []PackageVersion {
	var results []PackageVersion

	for packageName, currentVersion := range dependencies {
		// Skip if it's not a Laravel package
		if !IsLaravelPackage(packageName) {
			continue
		}

		// Check if package should be excluded
		if constraint, exists := constraints[packageName]; exists && constraint.ExcludePackage {
			results = append(results, PackageVersion{
				Name:       packageName,
				Skipped:    true,
				SkipReason: "Package excluded by constraints",
			})
			continue
		}

		parts := strings.Split(packageName, "/")
		if len(parts) != 2 {
			results = append(results, PackageVersion{
				Name:       packageName,
				Skipped:    true,
				SkipReason: "Invalid package name format",
			})
			continue
		}

		vendor, pkg := parts[0], parts[1]
		latestVersion, err := h.getLatestVersion(vendor, pkg)

		if err != nil {
			results = append(results, PackageVersion{
				Name:       packageName,
				Skipped:    true,
				SkipReason: fmt.Sprintf("Failed to fetch version info: %v", err),
			})
			continue
		}

		// Remove version constraint operators from current version
		cleanCurrentVersion := strings.TrimLeft(currentVersion, "^~>=<!")

		results = append(results, PackageVersion{
			Name:           packageName,
			CurrentVersion: &cleanCurrentVersion,
			LatestVersion:  latestVersion,
			Registry:       "packagist",
		})
	}

	return results
}
