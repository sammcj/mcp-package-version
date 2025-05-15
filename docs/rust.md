# Rust Packages Support

This document explains how to use the Rust crate version checking functionality in the MCP Package Version server.

## Checking Rust Crate Versions

The MCP server provides a tool called `check_rust_versions` that allows you to check the latest versions of Rust crates from `Cargo.toml`.

### Example Usage

```json
{
  "name": "check_rust_versions",
  "arguments": {
    "dependencies": {
      "serde": "1.0.136",
      "tokio": "1.17.0",
      "rocket": "0.5.0-rc.1",
      "clap": { 
        "version": "3.1.0",
        "features": ["derive", "cargo"]
      },
      "log": {
        "version": "0.4.17",
        "optional": true,
        "default": false
      }
    }
  }
}
```

### Alternative Format

You can also provide dependencies as an array:

```json
{
  "name": "check_rust_versions",
  "arguments": {
    "dependencies": [
      {
        "name": "serde",
        "version": "1.0.136"
      },
      {
        "name": "tokio",
        "version": "1.17.0"
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
    "name": "serde",
    "currentVersion": "1.0.136",
    "latestVersion": "1.0.188",
    "registry": "crates.io"
  },
  {
    "name": "clap",
    "currentVersion": "3.1.0",
    "latestVersion": "4.4.10",
    "registry": "crates.io",
    "metadata": "{\"features\":[\"derive\",\"cargo\"]}"
  },
  {
    "name": "log", 
    "currentVersion": "0.4.17",
    "latestVersion": "0.4.20",
    "registry": "crates.io",
    "metadata": "{\"optional\":true,\"default\":false}"
  }
]
```

## Implementation Details

The Rust crate version checker:

1. Parses the dependencies provided in the request
2. For each crate, queries the crates.io API to retrieve the latest version
3. Returns a list of crates with their current and latest versions

The implementation uses caching to avoid making repeated requests to the crates.io API for the same crates within a short time period.

## API Reference

The Rust crate version checker uses the crates.io API to fetch the latest versions:

- API Endpoint: `https://crates.io/api/v1/crates/{crate_name}`
- Response includes the crate information and available versions
