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
	// PubDevAPIURL is the base URL for the pub.dev API
	PubDevAPIURL = "https://pub.dev/api/packages"
)

// DartHandler handles Dart package version checking
type DartHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewDartHandler creates a new Dart handler
func NewDartHandler(logger *logrus.Logger, cache *sync.Map) *DartHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &DartHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// DartPackageInfo represents information about a Dart package from the pub.dev API
type DartPackageInfo struct {
	Name     string               `json:"name"`
	Latest   DartPackageVersion   `json:"latest"`
	Versions []DartPackageVersion `json:"versions"`
}

// DartPackageVersion represents a version of a Dart package
type DartPackageVersion struct {
	Version     string `json:"version"`
	PubspecYaml string `json:"pubspec,omitempty"`
	Published   string `json:"published,omitempty"`
	Retracted   bool   `json:"retracted,omitempty"`
}

// DartDependency represents a dependency in a Dart pubspec.yaml file
type DartDependency struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	SDK         bool   `json:"sdk,omitempty"`
	Environment bool   `json:"environment,omitempty"`
}

// getLatestVersion gets the latest version of a Dart package
func (h *DartHandler) getLatestVersion(packageName string) (string, error) {
	// Check cache first
	if cachedVersion, ok := h.cache.Load(fmt.Sprintf("dart:%s", packageName)); ok {
		h.logger.WithField("package", packageName).Debug("Using cached Dart package version")
		return cachedVersion.(string), nil
	}

	// Skip SDK or environment dependencies
	if packageName == "flutter" || packageName == "dart" || packageName == "flutter_test" || packageName == "flutter_driver" {
		return "sdk", nil
	}

	// Construct URL
	packageURL := fmt.Sprintf("%s/%s", PubDevAPIURL, packageName)
	h.logger.WithFields(logrus.Fields{
		"package": packageName,
		"url":     packageURL,
	}).Debug("Fetching Dart package info")

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", packageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Dart package info: %w", err)
	}

	// Parse response
	var info DartPackageInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("failed to parse Dart package info: %w", err)
	}

	// Get latest non-retracted version
	var latestVersion string
	if !info.Latest.Retracted {
		latestVersion = info.Latest.Version
	} else {
		// If latest is retracted, find the latest non-retracted version
		var validVersions []string
		for _, version := range info.Versions {
			if !version.Retracted {
				validVersions = append(validVersions, version.Version)
			}
		}

		if len(validVersions) == 0 {
			return "", fmt.Errorf("no valid versions found for package %s", packageName)
		}

		// Sort versions and get the latest
		sort.Slice(validVersions, func(i, j int) bool {
			cmp, err := CompareVersions(validVersions[i], validVersions[j])
			if err != nil {
				h.logger.WithError(err).WithFields(logrus.Fields{
					"version1": validVersions[i],
					"version2": validVersions[j],
				}).Error("Failed to compare versions")
				return false
			}
			return cmp > 0
		})

		latestVersion = validVersions[0]
	}

	// Cache result
	h.cache.Store(fmt.Sprintf("dart:%s", packageName), latestVersion)

	return latestVersion, nil
}

// GetLatestVersion gets the latest versions of Dart packages
func (h *DartHandler) GetLatestVersion(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	h.logger.Debug("Getting latest Dart package versions")

	// Parse dependencies
	depsRaw, ok := args["dependencies"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: dependencies")
	}

	// Handle different input formats
	var dependencies []DartDependency

	// Log the raw dependencies for debugging
	h.logger.WithField("dependencies", fmt.Sprintf("%+v", depsRaw)).Debug("Raw dependencies")

	// Handle dependencies as a map (like in pubspec.yaml)
	if depsMap, ok := depsRaw.(map[string]interface{}); ok {
		for name, versionRaw := range depsMap {
			h.logger.WithFields(logrus.Fields{
				"name":    name,
				"version": versionRaw,
			}).Debug("Processing dependency")

			// Identify if it's an SDK dependency
			isSDK := false
			isEnv := false

			// Check if the package name indicates it's from an SDK
			if name == "flutter" || name == "dart" || strings.HasPrefix(name, "flutter:") || strings.HasPrefix(name, "dart:") {
				isSDK = true
			}

			// Check if it's an environment dependency (typically starts with "sdk:")
			if strings.HasPrefix(name, "sdk:") {
				isEnv = true
			}

			// Handle version as a string
			if versionStr, ok := versionRaw.(string); ok {
				dependencies = append(dependencies, DartDependency{
					Name:        name,
					Version:     versionStr,
					SDK:         isSDK,
					Environment: isEnv,
				})
			} else if versionMap, ok := versionRaw.(map[string]interface{}); ok {
				// Handle version as an object
				var version string
				if v, ok := versionMap["version"].(string); ok {
					version = v
				} else if v, ok := versionMap["path"].(string); ok {
					// Path dependency
					version = fmt.Sprintf("path: %s", v)
				} else if v, ok := versionMap["git"].(map[string]interface{}); ok {
					// Git dependency
					url, _ := v["url"].(string)
					ref, _ := v["ref"].(string)
					if ref != "" {
						version = fmt.Sprintf("git: %s@%s", url, ref)
					} else {
						version = fmt.Sprintf("git: %s", url)
					}
				}

				dependencies = append(dependencies, DartDependency{
					Name:        name,
					Version:     version,
					SDK:         isSDK,
					Environment: isEnv,
				})
			}
		}
	} else if depsArray, ok := depsRaw.([]interface{}); ok {
		// Handle dependencies as an array of objects
		for _, depRaw := range depsArray {
			if depMap, ok := depRaw.(map[string]interface{}); ok {
				var dep DartDependency

				if name, ok := depMap["name"].(string); ok {
					dep.Name = name
				} else {
					continue
				}

				if version, ok := depMap["version"].(string); ok {
					dep.Version = version
				}

				// Check if it's an SDK dependency
				if sdk, ok := depMap["sdk"].(bool); ok {
					dep.SDK = sdk
				} else if strings.HasPrefix(dep.Name, "flutter:") || strings.HasPrefix(dep.Name, "dart:") ||
					dep.Name == "flutter" || dep.Name == "dart" {
					dep.SDK = true
				}

				// Check if it's an environment dependency
				if env, ok := depMap["environment"].(bool); ok {
					dep.Environment = env
				} else if strings.HasPrefix(dep.Name, "sdk:") {
					dep.Environment = true
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
			"package": dep.Name,
			"version": dep.Version,
			"sdk":     dep.SDK,
			"env":     dep.Environment,
		}).Debug("Processing Dart package")

		// Skip SDK or environment dependencies
		if dep.SDK || dep.Environment {
			results = append(results, PackageVersion{
				Name:           dep.Name,
				CurrentVersion: StringPtr(dep.Version),
				LatestVersion:  "sdk dependency",
				Registry:       "pub.dev",
				Skipped:        true,
				SkipReason:     "SDK or environment dependency, version is managed by the SDK",
			})
			continue
		}

		// Skip dependencies with special formats (git, path)
		if strings.HasPrefix(dep.Version, "git:") || strings.HasPrefix(dep.Version, "path:") {
			results = append(results, PackageVersion{
				Name:           dep.Name,
				CurrentVersion: StringPtr(dep.Version),
				LatestVersion:  "special dependency",
				Registry:       "pub.dev",
				Skipped:        true,
				SkipReason:     "Git or path dependency, not a version constraint",
			})
			continue
		}

		// Get latest version
		latestVersion, err := h.getLatestVersion(dep.Name)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"package": dep.Name,
				"error":   err.Error(),
			}).Error("Failed to get Dart package info")
			results = append(results, PackageVersion{
				Name:           dep.Name,
				CurrentVersion: StringPtr(dep.Version),
				LatestVersion:  "unknown",
				Registry:       "pub.dev",
				Skipped:        true,
				SkipReason:     fmt.Sprintf("Failed to fetch package info: %v", err),
			})
			continue
		}

		// Add result
		results = append(results, PackageVersion{
			Name:           dep.Name,
			CurrentVersion: StringPtr(dep.Version),
			LatestVersion:  latestVersion,
			Registry:       "pub.dev",
		})
	}

	// Sort results by name
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	return NewToolResultJSON(results)
}
