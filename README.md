# Package Version MCP Server

[![smithery badge](https://smithery.ai/badge/mcp-package-version)](https://smithery.ai/server/mcp-package-version)

An MCP server that provides tools for checking latest stable package versions from multiple package registries:

- npm (Node.js/JavaScript)
- PyPI (Python)
- Maven Central (Java)
- Go Proxy (Go)

This server helps LLMs ensure they're recommending up-to-date package versions when writing code.

<a href="https://glama.ai/mcp/servers/zkts2w92ba"><img width="380" height="200" src="https://glama.ai/mcp/servers/zkts2w92ba/badge" alt="https://github.com/sammcj/mcp-package-version MCP server" /></a>

## Screenshot

![tooling with and without mcp-package-version](images/with-without.jpg)

## Running

### Installing via Smithery

To install Package Version for Claude Desktop automatically via [Smithery](https://smithery.ai/server/mcp-package-version):

```bash
npx -y @smithery/cli install mcp-package-version --client claude
```

**Configure MCP Settings**

Add the following to your MCP settings file:

```json
{
  "mcpServers": {
    "package-version": {
      "command": "npx",
      "args": ["-y", "mcp-package-version"]
    }
  }
}
```

If you are behind a corporate proxy which MITMs your traffic, you may need to additionally specify the proxy CA cert bundle:
```json
{
  "mcpServers": {
    "package-version": {
      "command": "npx",
      "args": ["-y", "mcp-package-version"],
      "env": {
        "NODE_EXTRA_CA_CERTS": "/path/to/mitm/cert.pem"
      }
    }
  }
}
```

- For the Cline VSCode Extension this will be `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`
- For Claude Desktop `~/Library/Application\ Support/Claude/claude_desktop_config.json`
- For GoMCP `~/.config/gomcp/config.yaml`


## Tools

### 1. JavaScript/Node.js

#### check_npm_versions

Check latest stable versions for npm packages from a package.json dependencies object.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_npm_versions",
  arguments: {
    dependencies: {
      "express": "^4.17.1",
      "react": "^17.0.2"
    }
  }
});
```

### 2. Python

#### check_python_versions

Check latest stable versions for Python packages from requirements.txt entries.

#### check_pyproject_versions

Check latest stable versions for Python packages from pyproject.toml.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_pyproject_versions",
  arguments: {
    dependencies: {
      dependencies: {
        "requests": "^2.28.0",
        "pandas": ">=1.5.0"
      },
      "optional-dependencies": {
        "test": {
          "pytest": ">=7.0.0"
        }
      },
      "dev-dependencies": {
        "black": "^22.0.0"
      }
    }
  }
});
```

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_python_versions",
  arguments: {
    requirements: [
      "requests==2.26.0",
      "pandas>=1.3.0"
    ]
  }
});
```

### 3. Go

#### check_go_versions

Check latest stable versions for Go packages from go.mod.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_go_versions",
  arguments: {
    dependencies: {
      module: "example.com/mymodule",
      require: [
        {
          path: "github.com/gin-gonic/gin",
          version: "v1.7.0"
        }
      ],
      replace: [
        {
          old: "github.com/old/pkg",
          new: "github.com/new/pkg",
          version: "v2.0.0"
        }
      ]
    }
  }
});
```

### 4. Java

But seriously, don't write Java in 2025.

#### check_maven_versions

Check latest stable versions for Java packages from pom.xml.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_maven_versions",
  arguments: {
    dependencies: [
      {
        groupId: "org.springframework.boot",
        artifactId: "spring-boot-starter-web",
        version: "2.7.0",
        scope: "compile"
      }
    ]
  }
});
```

#### check_gradle_versions

Check latest stable versions for Java packages from build.gradle.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_gradle_versions",
  arguments: {
    dependencies: [
      {
        configuration: "implementation",
        group: "com.google.guava",
        name: "guava",
        version: "31.0-jre"
      }
    ]
  }
});
```

### 5. check_package_versions

Bulk check latest stable versions for multiple packages from npm and PyPI.

```typescript
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_package_versions",
  arguments: {
    packages: [
      { name: "react", registry: "npm" },
      { name: "requests", registry: "pypi" },
      { name: "typescript", registry: "npm", currentVersion: "5.0.0" }
    ]
  }
});
```

## Guidelines for LLMs

When writing code that includes package dependencies, LLMs should:

0. **Choose the Right Tool for the Job**
   - Use language-specific tools for detailed dependency management:
     - `check_npm_versions` for package.json
     - `check_python_versions` for requirements.txt
     - `check_pyproject_versions` for pyproject.toml
     - `check_maven_versions` for pom.xml
     - `check_gradle_versions` for build.gradle
     - `check_go_versions` for go.mod
   - Use `check_package_versions` for quick bulk checks across npm and PyPI

1. **Always Check Versions Before Writing**
   - Before writing a package.json or requirements.txt file, use the appropriate tool to check latest versions
   - Use the bulk check tool when dealing with multiple packages
   - Consider the project's needs when deciding whether to use exact versions or version ranges

2. **Package.json Best Practices**
   ```typescript
   // Before writing package.json, check versions
   const versions = await use_mcp_tool({
     server_name: "package-version",
     tool_name: "check_package_versions",
     arguments: {
       packages: [
         { name: "express", registry: "npm" },
         { name: "react", registry: "npm" }
       ]
     }
   });

   // Use the returned versions in package.json
   {
     "dependencies": {
       "express": "^{express.latestVersion}",
       "react": "^{react.latestVersion}"
     }
   }
   ```

3. **Requirements.txt Best Practices**
   ```typescript
   // Before writing requirements.txt, check versions
   const versions = await use_mcp_tool({
     server_name: "package-version",
     tool_name: "check_package_versions",
     arguments: {
       packages: [
         { name: "requests", registry: "pypi" },
         { name: "pandas", registry: "pypi" }
       ]
     }
   });

   // Use the returned versions in requirements.txt
   requests=={requests.latestVersion}
   pandas=={pandas.latestVersion}
   ```

4. **Version Range Considerations**
   - For applications: Consider using exact versions (= for Python, no prefix for npm)
   - For libraries: Consider using compatible ranges (>= for Python, ^ for npm)
   - Always document version choices in comments

5. **Error Handling**
   - If version check fails for a package, document it in comments
   - Consider falling back to known stable versions if checks fail
   - Warn users about any packages that couldn't be verified

## Example Integrations

Here's how an LLM should approach creating new projects with different package managers:

### Node.js Project

```typescript
// 1. Check npm package versions
const versions = await use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_npm_versions",
  arguments: {
    dependencies: {
      "express": "^4.17.1",
      "typescript": "~4.5.0"
    }
  }
});

// 2. Use the versions in package.json
write_to_file({

  path: "package.json",
  content: {
    "name": "my-project",
    "version": "1.0.0",
    "dependencies": {
      "express": `^${versions.find(p => p.name === 'express').latestVersion}`,
      "typescript": `^${versions.find(p => p.name === 'typescript').latestVersion}`
    }
  }
});
```

### Python Project with pyproject.toml

```typescript
// 1. Check Python package versions
const versions = await use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_pyproject_versions",
  arguments: {
    dependencies: {
      dependencies: {
        "requests": "^2.28.0",
        "pandas": ">=1.5.0"
      },
      "dev-dependencies": {
        "pytest": ">=7.0.0"
      }
    }
  }
});

// 2. Use the versions in pyproject.toml
write_to_file({
  path: "pyproject.toml",
  content: `
[project]
name = "my-project"
version = "1.0.0"
dependencies = [
    "requests>=${versions.find(p => p.name === 'requests').latestVersion}",
    "pandas>=${versions.find(p => p.name === 'pandas').latestVersion}"
]

[project.optional-dependencies]
test = [
    "pytest>=${versions.find(p => p.name === 'pytest (dev)').latestVersion}"
]
`
});
```

### Go Project

```typescript
// 1. Check Go package versions
const versions = await use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_go_versions",
  arguments: {
    dependencies: {
      module: "example.com/mymodule",
      require: [
        {
          path: "github.com/gin-gonic/gin",
          version: "v1.7.0"
        }
      ]
    }
  }
});

// 2. Use the versions in go.mod
write_to_file({
  path: "go.mod",
  content: `
module example.com/mymodule

go 1.21

require (
    github.com/gin-gonic/gin ${versions.find(p => p.name === 'github.com/gin-gonic/gin').latestVersion}
)
`
});
```

### Java Project with Maven

```typescript
// 1. Check Maven package versions
const versions = await use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_maven_versions",
  arguments: {
    dependencies: [
      {
        groupId: "org.springframework.boot",
        artifactId: "spring-boot-starter-web",
        version: "2.7.0"
      }
    ]
  }
});

// 2. Use the versions in pom.xml
write_to_file({
  path: "pom.xml",
  content: `
<project>
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>my-project</artifactId>
    <version>1.0.0</version>

    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
            <version>${versions.find(p => p.name === 'org.springframework.boot:spring-boot-starter-web').latestVersion}</version>
        </dependency>
    </dependencies>
</project>
`
});
```

This ensures that new projects always start with the latest stable versions of packages.

---

## Example System Prompt

```plaintext
When writing code that includes package dependencies, you must:

1. Before writing any dependency files, check latest stable versions using the package-version MCP server:

// For Node.js (package.json):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_npm_versions",
  arguments: {
    dependencies: {
      "package-name": "version"
    }
  }
});

// For Python (requirements.txt):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_python_versions",
  arguments: {
    requirements: [
      "package-name==version"
    }
  }
});

// For Python (pyproject.toml):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_pyproject_versions",
  arguments: {
    dependencies: {
      dependencies: {
        "package-name": "version"
      }
    }
  }
});

// For Java Maven (pom.xml):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_maven_versions",
  arguments: {
    dependencies: [
      {
        groupId: "group",
        artifactId: "artifact",
        version: "version"
      }
    ]
  }
});

// For Java Gradle (build.gradle):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_gradle_versions",
  arguments: {
    dependencies: [
      {
        configuration: "implementation",
        group: "group",
        name: "name",
        version: "version"
      }
    ]
  }
});

// For Go (go.mod):
use_mcp_tool({
  server_name: "package-version",
  tool_name: "check_go_versions",
  arguments: {
    dependencies: {
      module: "module-name",
      require: [
        {
          path: "package-path",
          version: "version"
        }
      ]
    }
  }
});

2. Use the returned latest versions in your dependency files:
   - For applications: Use exact versions
   - For libraries: Use compatible ranges
     - npm: ^ for minor updates, ~ for patch updates
     - Python: >= for compatible versions, == for exact versions
     - Java: Use the version directly (Maven/Gradle handle ranges differently)
     - Go: Use semantic version prefixes (e.g., v1.2.3)
   - Document any version-specific requirements in comments

3. If version checks fail:
   - Document it in comments
   - Use known stable versions as fallback
   - Consider project requirements and compatibility
```

Example system prompt for users:

```plaintext
When writing code that includes dependencies, you must check latest stable versions using the package-version MCP server before writing any dependency files (package.json, requirements.txt, pyproject.toml, pom.xml, build.gradle, go.mod). Use exact versions for applications and appropriate version ranges for libraries based on the package manager's conventions. Document any version-specific requirements or failed checks in comments.
```

## Development

1. **Clone and Install Dependencies**
   ```bash
   git clone https://github.com/sammcj/mcp-package-version.git
   cd mcp-package-version
   npm i
   ```

2. **Build the Server**
   ```bash
   npm run build
   ```

3. **Development Workflow**
   - Use `npm run watch` for development to automatically rebuild on changes
   - Use `npm run build` for production builds

4. **Release Process**
   ```bash
   # 1. Make your changes
   vim src/your-file.ts

   # 2. Commit your changes
   git add .
   git commit -m "feat: your new feature"

   # 3. Run bump command (this will):
   # - Update version in package.json
   # - Update CHANGELOG.md
   # - Commit changes
   # - Push to GitHub
   npm run bump

   # GitHub Actions will then:
   # - Create a git tag
   # - Create a GitHub release
   # - Publish to npm (when triggered manually)
   ```

5. **Manual npm Publishing**
   ```bash
   # To trigger a manual npm publish
   gh workflow run publish.yml
   ```

No environment variables are required as this server uses public registries:

- npm registry (registry.npmjs.org)
- PyPI (pypi.org)
- Go Proxy (proxy.golang.org)
- Maven Central (search.maven.org)

## License

[MIT](LICENSE)
