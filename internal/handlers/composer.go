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

// ComposerHandler handles Composer package version checking for PHP packages
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
	packagistAPIURL     = "https://packagist.org/p2/%s/%s.json"
	packagistSearchURL  = "https://packagist.org/search.json?q=%s"
	packagistPackageURL = "https://packagist.org/packages/%s/%s.json"
	packagistMetaURL    = "https://repo.packagist.org/p2/%s/%s.json" // Alternative API endpoint
)

var (
	laravelVendors = map[string]bool{
		"illuminate": true,
		"spatie":     true,
		"filament":   true,
		"laravel":    true, // Add the laravel vendor
	}
)

// GetLatestVersion gets information about Composer package versions
func (h *ComposerHandler) GetLatestVersion(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	h.logger.Debug("Getting Composer package version information")

	// Parse dependencies
	dependenciesRaw, ok := args["dependencies"].(map[string]interface{})
	if !ok {
		h.logger.Error("Missing required parameter: dependencies")
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

	var results []PackageVersion

	// Process each dependency
	for packageName, currentVersion := range dependencies {
		h.logger.WithFields(logrus.Fields{
			"package": packageName,
			"version": currentVersion,
		}).Debug("Checking package")

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
		latestVersion, err := h.fetchLatestVersion(vendor, pkg)

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

// PackagistSearchResponse represents a response from the Packagist search API
type PackagistSearchResponse struct {
	Results []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Repository  string `json:"repository"`
		Downloads   int    `json:"downloads"`
		Favers      int    `json:"favers"`
	} `json:"results"`
	Total int `json:"total"`
}

// PackagistPackageResponse represents a response from the Packagist package API
type PackagistPackageResponse struct {
	Package struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Time        string `json:"time"`
		Maintainers []struct {
			Name string `json:"name"`
		} `json:"maintainers"`
		Versions map[string]struct {
			Version     string            `json:"version"`
			VersionNorm string            `json:"version_normalized"`
			Time        time.Time         `json:"time"`
			License     []string          `json:"license"`
			Description string            `json:"description"`
			Homepage    string            `json:"homepage"`
			Keywords    []string          `json:"keywords"`
			Type        string            `json:"type"`
			RequiresPHP string            `json:"require-php"`
			RequiresDev map[string]string `json:"require-dev"`
			Requires    map[string]string `json:"require"`
			Authors     []struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Homepage string `json:"homepage"`
				Role     string `json:"role"`
			} `json:"authors"`
		} `json:"versions"`
	} `json:"package"`
}

// fetchLatestVersion retrieves the latest version of a package from Packagist
func (h *ComposerHandler) fetchLatestVersion(vendor, package_ string) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("packagist:%s/%s", vendor, package_)
	if cachedVersion, ok := h.cache.Load(cacheKey); ok {
		h.logger.WithField("package", fmt.Sprintf("%s/%s", vendor, package_)).Debug("Using cached version")
		return cachedVersion.(string), nil
	}

	// Try different API endpoints
	endpoints := []string{
		fmt.Sprintf(packagistPackageURL, vendor, package_),
		fmt.Sprintf(packagistAPIURL, vendor, package_),
		fmt.Sprintf(packagistMetaURL, vendor, package_),
	}

	var body []byte
	var err error
	var succeeded bool

	for _, url := range endpoints {
		h.logger.WithFields(logrus.Fields{
			"package": fmt.Sprintf("%s/%s", vendor, package_),
			"url":     url,
		}).Debug("Trying API endpoint")

		body, err = MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
		if err == nil {
			succeeded = true
			break
		}
		h.logger.WithError(err).WithField("url", url).Debug("Failed to fetch from endpoint, trying next")
	}

	if !succeeded {
		// If all direct endpoints fail, try the search API
		searchURL := fmt.Sprintf(packagistSearchURL, fmt.Sprintf("%s/%s", vendor, package_))
		h.logger.WithField("url", searchURL).Debug("Trying search API")

		searchBody, searchErr := MakeRequestWithLogger(h.client, h.logger, "GET", searchURL, nil)
		if searchErr != nil {
			h.logger.WithError(searchErr).WithField("url", searchURL).Error("Failed to search for package")
			return "", fmt.Errorf("failed to search for package: %v", searchErr)
		}

		var searchResponse PackagistSearchResponse
		if err := json.Unmarshal(searchBody, &searchResponse); err != nil {
			h.logger.WithError(err).Error("Failed to parse search response")
			return "", fmt.Errorf("failed to parse search response: %v", err)
		}

		if searchResponse.Total == 0 || len(searchResponse.Results) == 0 {
			h.logger.Error("No packages found in search results")
			return "", fmt.Errorf("no packages found for %s/%s", vendor, package_)
		}

		// Find the exact package match
		var exactMatch bool
		for _, result := range searchResponse.Results {
			if strings.EqualFold(result.Name, fmt.Sprintf("%s/%s", vendor, package_)) {
				exactMatch = true
				break
			}
		}

		if !exactMatch {
			h.logger.WithField("results", fmt.Sprintf("%+v", searchResponse.Results)).Error("No exact match found in search results")
			return "", fmt.Errorf("no exact match found for %s/%s", vendor, package_)
		}

		// Try the package URL again with the confirmed package name
		url := fmt.Sprintf(packagistPackageURL, vendor, package_)
		body, err = MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
		if err != nil {
			h.logger.WithError(err).Error("Failed to fetch package info after confirming existence")
			return "", fmt.Errorf("failed to fetch package info: %v", err)
		}
	}

	// Parse the response
	var packageResponse PackagistPackageResponse
	if err := json.Unmarshal(body, &packageResponse); err != nil {
		h.logger.WithError(err).Error("Failed to parse package info")
		return "", fmt.Errorf("failed to parse package info: %v", err)
	}

	var latestVersion string
	var latestTime time.Time

	for versionStr, versionInfo := range packageResponse.Package.Versions {
		// Skip dev versions
		if strings.Contains(versionStr, "dev-") || strings.Contains(versionStr, "-dev") {
			continue
		}

		if versionInfo.Time.After(latestTime) {
			latestTime = versionInfo.Time
			latestVersion = versionInfo.Version
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no stable versions found for %s/%s", vendor, package_)
	}

	// Cache the result
	h.cache.Store(cacheKey, latestVersion)

	return latestVersion, nil
}
