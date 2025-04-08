# Package Version MCP Server

[![smithery badge](https://smithery.ai/badge/mcp-package-version)](https://smithery.ai/server/mcp-package-version)

An MCP server that provides tools for checking latest stable package versions from multiple package registries:

- npm (Node.js/JavaScript)
- PyPI (Python)
- Maven Central (Java)
- Go Proxy (Go)
- Swift Packages (Swift)
- AWS Bedrock (AI Models)
- Docker Hub (Container Images)
- GitHub Container Registry (Container Images)
- GitHub Actions

This server helps LLMs ensure they're recommending up-to-date package versions when writing code.

**IMPORTANT: As of version 2.0.0, mcp-package-version has been rewritten in Go, as such the configuration needs to be updated in your client - see the [Installation](#installation) section for more details.**

<a href="https://glama.ai/mcp/servers/zkts2w92ba"><img width="380" height="200" src="https://glama.ai/mcp/servers/zkts2w92ba/badge" alt="https://github.com/sammcj/mcp-package-version MCP server" /></a>

## Screenshot

![tooling with and without mcp-package-version](images/with-without.jpg)

## Installation

```bash
go install github.com/sammcj/mcp-package-version@HEAD
```

Then setup your client to use the MCP server

- For the Cline VSCode Extension this will be `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`
- For Claude Desktop `~/Library/Application\ Support/Claude/claude_desktop_config.json`
- For GoMCP `~/.config/gomcp/config.yaml`

Assuming you've installed the binary with `go install github.com/sammcj/mcp-package-version@HEAD` and your `$GOPATH` is `/Users/sam/go/bin`, you can provide the full path to the binary:

```json
{
  "mcpServers": {
    "package-version": {
      "command": "/Users/sam/go/bin/mcp-package-version"
    }
  }
}
```

### Other Installation Methods

Using go run:

```json
{
  "mcpServers": {
    "package-version": {
        "command": "go run github.com/sammcj/gollama@HEAD",
      }
  }
}
```

Installing a specific version:

```bash
go install github.com/sammcj/mcp-package-version@v2.0.0
```

Or clone the repository and build it:

```bash
git clone https://github.com/sammcj/mcp-package-version.git
cd mcp-package-version
make
```

You can also run the server in a container:

```bash
docker run -p 18080:18080 ghcr.io/sammcj/mcp-package-version:latest
```

Note: If running in a container, you'll need to configure the client to use the URL instead of command, e.g.:

```json
{
  "mcpServers": {
    "package-version": {
      "url": "http://localhost:18080",
    }
  }
}
```

### Version Information

You can check the version of the installed binary:

```bash
mcp-package-version version
```

## Usage

The server supports two transport modes: stdio (default) and SSE (Server-Sent Events).

### STDIO Transport (Default)

```bash
mcp-package-version
```

Or if you built it locally:

```bash
./bin/mcp-package-version
```

### SSE Transport

```bash
mcp-package-version --transport sse --port 18080 --base-url http://localhost
```

Or if you built it locally:

```bash
./bin/mcp-package-version --transport sse --port 18080 --base-url http://localhost
```

#### Command-line Options

- `--transport`, `-t`: Transport type (stdio or sse). Default: stdio
- `--port`: Port to use for SSE transport. Default: 18080
- `--base-url`: Base URL for SSE transport. Default: http://localhost

## Tools

### NPM Packages

Check the latest versions of NPM packages:

```json
{
  "name": "check_npm_versions",
  "arguments": {
    "dependencies": {
      "react": "^17.0.2",
      "react-dom": "^17.0.2",
      "lodash": "4.17.21"
    },
    "constraints": {
      "react": {
        "majorVersion": 17
      }
    }
  }
}
```

### Python Packages (requirements.txt)

Check the latest versions of Python packages from requirements.txt:

```json
{
  "name": "check_python_versions",
  "arguments": {
    "requirements": [
      "requests==2.28.1",
      "flask>=2.0.0",
      "numpy"
    ]
  }
}
```

### Python Packages (pyproject.toml)

Check the latest versions of Python packages from pyproject.toml:

```json
{
  "name": "check_pyproject_versions",
  "arguments": {
    "dependencies": {
      "dependencies": {
        "requests": "^2.28.1",
        "flask": ">=2.0.0"
      },
      "optional-dependencies": {
        "dev": {
          "pytest": "^7.0.0"
        }
      },
      "dev-dependencies": {
        "black": "^22.6.0"
      }
    }
  }
}
```

### Java Packages (Maven)

Check the latest versions of Java packages from Maven:

```json
{
  "name": "check_maven_versions",
  "arguments": {
    "dependencies": [
      {
        "groupId": "org.springframework.boot",
        "artifactId": "spring-boot-starter-web",
        "version": "2.7.0"
      },
      {
        "groupId": "com.google.guava",
        "artifactId": "guava",
        "version": "31.1-jre"
      }
    ]
  }
}
```

### Java Packages (Gradle)

Check the latest versions of Java packages from Gradle:

```json
{
  "name": "check_gradle_versions",
  "arguments": {
    "dependencies": [
      {
        "configuration": "implementation",
        "group": "org.springframework.boot",
        "name": "spring-boot-starter-web",
        "version": "2.7.0"
      },
      {
        "configuration": "testImplementation",
        "group": "junit",
        "name": "junit",
        "version": "4.13.2"
      }
    ]
  }
}
```

### Go Packages

Check the latest versions of Go packages from go.mod:

```json
{
  "name": "check_go_versions",
  "arguments": {
    "dependencies": {
      "module": "github.com/example/mymodule",
      "require": [
        {
          "path": "github.com/gorilla/mux",
          "version": "v1.8.0"
        },
        {
          "path": "github.com/spf13/cobra",
          "version": "v1.5.0"
        }
      ]
    }
  }
}
```

### Docker Images

Check available tags for Docker images:

```json
{
  "name": "check_docker_tags",
  "arguments": {
    "image": "nginx",
    "registry": "dockerhub",
    "limit": 5,
    "filterTags": ["^1\\."],
    "includeDigest": true
  }
}
```

### AWS Bedrock Models

List all AWS Bedrock models:

```json
{
  "name": "check_bedrock_models",
  "arguments": {
    "action": "list"
  }
}
```

Search for specific AWS Bedrock models:

```json
{
  "name": "check_bedrock_models",
  "arguments": {
    "action": "search",
    "query": "claude",
    "provider": "anthropic"
  }
}
```

Get the latest Claude Sonnet model:

```json
{
  "name": "get_latest_bedrock_model",
  "arguments": {}
}
```

### Swift Packages

Check the latest versions of Swift packages:

```json
{
  "name": "check_swift_versions",
  "arguments": {
    "dependencies": [
      {
        "url": "https://github.com/apple/swift-argument-parser",
        "version": "1.1.4"
      },
      {
        "url": "https://github.com/vapor/vapor",
        "version": "4.65.1"
      }
    ],
    "constraints": {
      "https://github.com/apple/swift-argument-parser": {
        "majorVersion": 1
      }
    }
  }
}
```

### GitHub Actions

Check the latest versions of GitHub Actions:

```json
{
  "name": "check_github_actions",
  "arguments": {
    "actions": [
      {
        "owner": "actions",
        "repo": "checkout",
        "currentVersion": "v3"
      },
      {
        "owner": "actions",
        "repo": "setup-node",
        "currentVersion": "v3"
      }
    ],
    "includeDetails": true
  }
}
```

## Releases and CI/CD

This project uses GitHub Actions for continuous integration and deployment. The workflow automatically:

1. Builds and tests the application on every push to the main branch and pull requests
2. Creates a release when a tag with the format `v*` (e.g., `v1.0.0`) is pushed
3. Builds and pushes Docker images to GitHub Container Registry

### Docker Images

Docker images are available from GitHub Container Registry:

```bash
docker pull ghcr.io/sammcj/mcp-package-version:latest
```

Or with a specific version:

```bash
docker pull ghcr.io/sammcj/mcp-package-version:v2.0.0
```

## License

[MIT](LICENSE)
