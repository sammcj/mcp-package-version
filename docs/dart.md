# Dart Packages Support

This document explains how to use the Dart package version checking functionality in the MCP Package Version server.

## Checking Dart Package Versions

The MCP server provides a tool called `check_dart_versions` that allows you to check the latest versions of Dart packages from `pubspec.yaml`.

### Example Usage

```json
{
  "name": "check_dart_versions",
  "arguments": {
    "dependencies": {
      "flutter": "sdk: flutter",
      "http": "^0.13.4",
      "provider": "^6.0.2",
      "path": "^1.8.0",
      "firebase_core": "^1.12.0",
      "shared_preferences": {
        "version": "^2.0.13"
      },
      "myproject": {
        "path": "../myproject"
      },
      "flutter_auth": {
        "git": {
          "url": "https://github.com/example/flutter_auth.git",
          "ref": "main"
        }
      }
    }
  }
}
```

### Alternative Format

You can also provide dependencies as an array:

```json
{
  "name": "check_dart_versions",
  "arguments": {
    "dependencies": [
      {
        "name": "http",
        "version": "^0.13.4"
      },
      {
        "name": "provider",
        "version": "^6.0.2"
      },
      {
        "name": "flutter",
        "version": "sdk: flutter",
        "sdk": true
      }
    ]
  }
}
```

### Response Format

The response will contain an array of packages with their current and latest versions:

```json
[
  {
    "name": "flutter",
    "currentVersion": "sdk: flutter",
    "latestVersion": "sdk dependency",
    "registry": "pub.dev",
    "skipped": true,
    "skipReason": "SDK or environment dependency, version is managed by the SDK"
  },
  {
    "name": "http",
    "currentVersion": "^0.13.4",
    "latestVersion": "1.1.0",
    "registry": "pub.dev"
  },
  {
    "name": "provider",
    "currentVersion": "^6.0.2",
    "latestVersion": "6.0.5",
    "registry": "pub.dev"
  },
  {
    "name": "myproject",
    "currentVersion": "path: ../myproject",
    "latestVersion": "special dependency",
    "registry": "pub.dev",
    "skipped": true,
    "skipReason": "Git or path dependency, not a version constraint"
  }
]
```

## Implementation Details

The Dart package version checker:

1. Parses the dependencies provided in the request
2. For each package, queries the pub.dev API to retrieve the latest version
3. Returns a list of packages with their current and latest versions

The implementation uses caching to avoid making repeated requests to the pub.dev API for the same packages within a short time period.

### Special Cases

The checker handles several special cases:

- **SDK dependencies**: Flutter SDK packages (flutter, dart, etc.) are marked as "sdk dependency" and skipped
- **Path dependencies**: Local path dependencies are skipped as they don't have a central registry version
- **Git dependencies**: Dependencies fetched from git repositories are skipped

## API Reference

The Dart package version checker uses the pub.dev API to fetch the latest versions:

- API Endpoint: `https://pub.dev/api/packages/{package_name}`
- Response includes the package information and available versions
