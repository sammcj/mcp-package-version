package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// CratesioAPIURL is the base URL for the crates.io API
	CratesioAPIURL = "https://crates.io/api/v1"
)

// RustHandler handles Rust package version checking
type RustHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewRustHandler creates a new Rust handler
func NewRustHandler(logger *logrus.Logger, cache *sync.Map) *RustHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &RustHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// RustCrateInfo represents information about a Rust crate from the crates.io API
type RustCrateInfo struct {
	Crate struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Description      string `json:"description"`
		MaxVersion       string `json:"max_version"`
		MaxStableVersion string `json:"max_stable_version"`
	} `json:"crate"`
	Versions []struct {
		ID      string `json:"id"`
		Version string `json:"num"`
		Yanked  bool   `json:"yanked"`
	} `json:"versions"`
}

// RustDependency represents a dependency in a Rust Cargo.toml file
type RustDependency struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	// Additional fields like features, optional, etc. could be added here
}

// getLatestVersion gets the latest version of a Rust crate
func (h *RustHandler) getLatestVersion(crateName string) (string, error) {
	// Check cache first
	if cachedVersion, ok := h.cache.Load(fmt.Sprintf("rust:%s", crateName)); ok {
		h.logger.WithField("crate", crateName).Debug("Using cached Rust crate version")
		return cachedVersion.(string), nil
	}

	// Construct URL
	crateURL := fmt.Sprintf("%s/crates/%s", CratesioAPIURL, crateName)
	h.logger.WithFields(logrus.Fields{
		"crate": crateName,
		"url":   crateURL,
	}).Debug("Fetching Rust crate info")

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", crateURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Rust crate info: %w", err)
	}

	// Parse response
	var info RustCrateInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("failed to parse Rust crate info: %w", err)
	}

	// Get latest stable version
	var latestVersion string
	if info.Crate.MaxStableVersion != "" {
		// Use the max stable version if available
		latestVersion = info.Crate.MaxStableVersion
	} else if info.Crate.MaxVersion != "" {
		// Fallback to max version if max stable is not available
		latestVersion = info.Crate.MaxVersion
	} else {
		// If neither is available, try to find the latest non-yanked version
		var latestVersions []string
		for _, version := range info.Versions {
			if !version.Yanked {
				latestVersions = append(latestVersions, version.Version)
			}
		}

		if len(latestVersions) == 0 {
			return "", fmt.Errorf("no valid versions found for crate %s", crateName)
		}

		// Sort versions and get the latest
		sort.Slice(latestVersions, func(i, j int) bool {
			cmp, err := CompareVersions(latestVersions[i], latestVersions[j])
			if err != nil {
				h.logger.WithError(err).WithFields(logrus.Fields{
					"version1": latestVersions[i],
					"version2": latestVersions[j],
				}).Error("Failed to compare versions")
				return false
			}
			return cmp > 0
		})

		latestVersion = latestVersions[0]
	}

	// Cache result
	h.cache.Store(fmt.Sprintf("rust:%s", crateName), latestVersion)

	return latestVersion, nil
}

// GetLatestVersion gets the latest versions of Rust packages
func (h *RustHandler) GetLatestVersion(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	h.logger.Debug("Getting latest Rust package versions")

	// Parse dependencies
	depsRaw, ok := args["dependencies"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: dependencies")
	}

	// Handle different input formats
	var dependencies []RustDependency

	// Log the raw dependencies for debugging
	h.logger.WithField("dependencies", fmt.Sprintf("%+v", depsRaw)).Debug("Raw dependencies")

	// Handle dependencies as a map (like in Cargo.toml)
	if depsMap, ok := depsRaw.(map[string]interface{}); ok {
		for name, versionRaw := range depsMap {
			h.logger.WithFields(logrus.Fields{
				"name":    name,
				"version": versionRaw,
			}).Debug("Processing dependency")

			// Handle version as a string
			if versionStr, ok := versionRaw.(string); ok {
				dependencies = append(dependencies, RustDependency{
					Name:    name,
					Version: versionStr,
				})
			} else if versionMap, ok := versionRaw.(map[string]interface{}); ok {
				// Handle version as an object (e.g., with features, etc.)
				var version string
				if v, ok := versionMap["version"].(string); ok {
					version = v
				}

				dependencies = append(dependencies, RustDependency{
					Name:    name,
					Version: version,
				})
			}
		}
	} else if depsArray, ok := depsRaw.([]interface{}); ok {
		// Handle dependencies as an array of objects
		for _, depRaw := range depsArray {
			if depMap, ok := depRaw.(map[string]interface{}); ok {
				var dep RustDependency

				if name, ok := depMap["name"].(string); ok {
					dep.Name = name
				} else {
					continue
				}

				if version, ok := depMap["version"].(string); ok {
					dep.Version = version
				}

				dependencies = append(dependencies, dep)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid dependencies format: expected object or array, got %T", depsRaw)
	}

	// Process each dependency
	results := make([]PackageVersion, 0, len(dependencies))
	for _, dep := range dependencies {
		h.logger.WithFields(logrus.Fields{
			"crate":   dep.Name,
			"version": dep.Version,
		}).Debug("Processing Rust crate")

		// Get latest version
		latestVersion, err := h.getLatestVersion(dep.Name)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"crate": dep.Name,
				"error": err.Error(),
			}).Error("Failed to get Rust crate info")
			results = append(results, PackageVersion{
				Name:           dep.Name,
				CurrentVersion: StringPtr(dep.Version),
				LatestVersion:  "unknown",
				Registry:       "crates.io",
				Skipped:        true,
				SkipReason:     fmt.Sprintf("Failed to fetch crate info: %v", err),
			})
			continue
		}

		// Add result
		results = append(results, PackageVersion{
			Name:           dep.Name,
			CurrentVersion: StringPtr(dep.Version),
			LatestVersion:  latestVersion,
			Registry:       "crates.io",
		})
	}

	// Sort results by name
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	return NewToolResultJSON(results)
}
