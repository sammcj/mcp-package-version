#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ErrorCode,
  ListToolsRequestSchema,
  McpError,
} from '@modelcontextprotocol/sdk/types.js'
import axios from 'axios'
// import * as semver from 'semver';

interface PackageVersion {
  name: string
  currentVersion?: string
  latestVersion: string
  registry: 'npm' | 'pypi' | 'maven'
}

interface PyProjectDependencies {
  dependencies?: { [key: string]: string }
  'optional-dependencies'?: { [key: string]: { [key: string]: string } }
  'dev-dependencies'?: { [key: string]: string }
}

interface MavenDependency {
  groupId: string
  artifactId: string
  version?: string
  scope?: string
}

interface GradleDependency {
  configuration: string
  group: string
  name: string
  version?: string
}

class PackageVersionServer {
  private server: Server
  private npmRegistry = 'https://registry.npmjs.org';
  private pypiRegistry = 'https://pypi.org/pypi';
  private mavenRegistry = 'https://search.maven.org/solrsearch/select';

  constructor() {
    this.server = new Server(
      {
        name: 'package-version-server',
        version: '0.1.0',
      },
      {
        capabilities: {
          tools: {},
        },
      }
    )

    this.setupToolHandlers()

    this.server.onerror = (error) => console.error('[MCP Error]', error)
    process.on('SIGINT', async () => {
      await this.server.close()
      process.exit(0)
    })
  }

  private setupToolHandlers() {
    this.server.setRequestHandler(ListToolsRequestSchema, async () => ({
      tools: [
        {
          name: 'check_maven_versions',
          description: 'Check latest stable versions for Java packages in pom.xml',
          inputSchema: {
            type: 'object',
            properties: {
              dependencies: {
                type: 'array',
                items: {
                  type: 'object',
                  properties: {
                    groupId: {
                      type: 'string',
                      description: 'Maven group ID',
                    },
                    artifactId: {
                      type: 'string',
                      description: 'Maven artifact ID',
                    },
                    version: {
                      type: 'string',
                      description: 'Current version (optional)',
                    },
                    scope: {
                      type: 'string',
                      description: 'Dependency scope (e.g., compile, test, provided)',
                    },
                  },
                  required: ['groupId', 'artifactId'],
                },
                description: 'Array of Maven dependencies',
              },
            },
            required: ['dependencies'],
          },
        },
        {
          name: 'check_gradle_versions',
          description: 'Check latest stable versions for Java packages in build.gradle',
          inputSchema: {
            type: 'object',
            properties: {
              dependencies: {
                type: 'array',
                items: {
                  type: 'object',
                  properties: {
                    configuration: {
                      type: 'string',
                      description: 'Gradle configuration (e.g., implementation, testImplementation)',
                    },
                    group: {
                      type: 'string',
                      description: 'Package group',
                    },
                    name: {
                      type: 'string',
                      description: 'Package name',
                    },
                    version: {
                      type: 'string',
                      description: 'Current version (optional)',
                    },
                  },
                  required: ['configuration', 'group', 'name'],
                },
                description: 'Array of Gradle dependencies',
              },
            },
            required: ['dependencies'],
          },
        },
        {
          name: 'check_pyproject_versions',
          description: 'Check latest stable versions for Python packages in pyproject.toml',
          inputSchema: {
            type: 'object',
            properties: {
              dependencies: {
                type: 'object',
                properties: {
                  dependencies: {
                    type: 'object',
                    additionalProperties: {
                      type: 'string',
                    },
                    description: 'Project dependencies from pyproject.toml',
                  },
                  'optional-dependencies': {
                    type: 'object',
                    additionalProperties: {
                      type: 'object',
                      additionalProperties: {
                        type: 'string',
                      },
                    },
                    description: 'Optional dependencies from pyproject.toml',
                  },
                  'dev-dependencies': {
                    type: 'object',
                    additionalProperties: {
                      type: 'string',
                    },
                    description: 'Development dependencies from pyproject.toml',
                  },
                },
                description: 'Dependencies object from pyproject.toml',
              },
            },
            required: ['dependencies'],
          },
        },
        {
          name: 'check_npm_versions',
          description: 'Check latest stable versions for npm packages',
          inputSchema: {
            type: 'object',
            properties: {
              dependencies: {
                type: 'object',
                additionalProperties: {
                  type: 'string',
                },
                description: 'Dependencies object from package.json',
              },
            },
            required: ['dependencies'],
          },
        },
        {
          name: 'check_python_versions',
          description: 'Check latest stable versions for Python packages',
          inputSchema: {
            type: 'object',
            properties: {
              requirements: {
                type: 'array',
                items: {
                  type: 'string',
                },
                description: 'Array of requirements from requirements.txt',
              },
            },
            required: ['requirements'],
          },
        },
        {
          name: 'check_package_versions',
          description: 'Bulk check latest stable versions for npm and Python packages',
          inputSchema: {
            type: 'object',
            properties: {
              packages: {
                type: 'array',
                items: {
                  type: 'object',
                  properties: {
                    name: {
                      type: 'string',
                      description: 'Package name',
                    },
                    registry: {
                      type: 'string',
                      enum: ['npm', 'pypi'],
                      description: 'Package registry (npm or pypi)',
                    },
                    currentVersion: {
                      type: 'string',
                      description: 'Current version (optional)',
                    },
                  },
                  required: ['name', 'registry'],
                },
                description: 'Array of packages to check',
              },
            },
            required: ['packages'],
          },
        },
      ],
    }))

    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      switch (request.params.name) {
        case 'check_maven_versions':
          return this.handleMavenVersionCheck(request.params.arguments)
        case 'check_gradle_versions':
          return this.handleGradleVersionCheck(request.params.arguments)
        case 'check_pyproject_versions':
          return this.handlePyProjectVersionCheck(request.params.arguments)
        case 'check_npm_versions':
          return this.handleNpmVersionCheck(request.params.arguments)
        case 'check_python_versions':
          return this.handlePythonVersionCheck(request.params.arguments)
        case 'check_package_versions':
          return this.handleBulkVersionCheck(request.params.arguments)
        default:
          throw new McpError(
            ErrorCode.MethodNotFound,
            `Unknown tool: ${request.params.name}`
          )
      }
    })
  }

  private async getNpmPackageVersion(
    packageName: string,
    currentVersion?: string
  ): Promise<PackageVersion> {
    try {
      const response = await axios.get(
        `${this.npmRegistry}/${encodeURIComponent(packageName)}`
      )

      const latestVersion = response.data['dist-tags']?.latest
      if (!latestVersion) {
        throw new Error('Latest version not found')
      }

      const result: PackageVersion = {
        name: packageName,
        latestVersion,
        registry: 'npm',
      }

      if (currentVersion) {
        // Remove any leading ^ or ~ from the current version
        const cleanCurrentVersion = currentVersion.replace(/^[\^~]/, '')
        result.currentVersion = cleanCurrentVersion
      }

      return result
    } catch (error) {
      console.error(`Error fetching npm package ${packageName}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch npm package ${packageName}`
      )
    }
  }

  private async getMavenPackageVersion(
    groupId: string,
    artifactId: string,
    currentVersion?: string,
    scope?: string
  ): Promise<PackageVersion> {
    try {
      const query = `g:"${groupId}" AND a:"${artifactId}"`
      const response = await axios.get(this.mavenRegistry, {
        params: {
          q: query,
          core: 'gav',
          rows: 1,
          wt: 'json',
        },
      })

      const doc = response.data?.response?.docs?.[0]
      if (!doc?.latestVersion) {
        throw new Error('Latest version not found')
      }

      const name = scope
        ? `${groupId}:${artifactId} (${scope})`
        : `${groupId}:${artifactId}`

      const result: PackageVersion = {
        name,
        latestVersion: doc.latestVersion,
        registry: 'maven',
      }

      if (currentVersion) {
        result.currentVersion = currentVersion
      }

      return result
    } catch (error) {
      console.error(`Error fetching Maven package ${groupId}:${artifactId}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch Maven package ${groupId}:${artifactId}`
      )
    }
  }

  private async handleMavenVersionCheck(args: any) {
    if (!args.dependencies || !Array.isArray(args.dependencies)) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid Maven dependencies array'
      )
    }

    const results: PackageVersion[] = []
    for (const dep of args.dependencies) {
      if (!dep.groupId || !dep.artifactId) continue

      try {
        const result = await this.getMavenPackageVersion(
          dep.groupId,
          dep.artifactId,
          dep.version,
          dep.scope
        )
        results.push(result)
      } catch (error) {
        console.error(`Error checking Maven package ${dep.groupId}:${dep.artifactId}:`, error)
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  private async handleGradleVersionCheck(args: any) {
    if (!args.dependencies || !Array.isArray(args.dependencies)) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid Gradle dependencies array'
      )
    }

    const results: PackageVersion[] = []
    for (const dep of args.dependencies) {
      if (!dep.group || !dep.name || !dep.configuration) continue

      try {
        const result = await this.getMavenPackageVersion(
          dep.group,
          dep.name,
          dep.version,
          dep.configuration
        )
        results.push(result)
      } catch (error) {
        console.error(`Error checking Gradle package ${dep.group}:${dep.name}:`, error)
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  private async getPyPiPackageVersion(
    packageName: string,
    currentVersion?: string
  ): Promise<PackageVersion> {
    try {
      const response = await axios.get(
        `${this.pypiRegistry}/${encodeURIComponent(packageName)}/json`
      )

      const latestVersion = response.data.info.version
      if (!latestVersion) {
        throw new Error('Latest version not found')
      }

      const result: PackageVersion = {
        name: packageName,
        latestVersion,
        registry: 'pypi',
      }

      if (currentVersion) {
        // Remove any comparison operators from the current version
        const cleanCurrentVersion = currentVersion.replace(/^[=<>~!]+/, '')
        result.currentVersion = cleanCurrentVersion
      }

      return result
    } catch (error) {
      console.error(`Error fetching PyPI package ${packageName}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch PyPI package ${packageName}`
      )
    }
  }

  private async handleNpmVersionCheck(args: any) {
    if (!args.dependencies || typeof args.dependencies !== 'object') {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid dependencies object'
      )
    }

    const results: PackageVersion[] = []
    for (const [name, version] of Object.entries(args.dependencies)) {
      if (typeof version !== 'string') continue
      try {
        const result = await this.getNpmPackageVersion(name, version)
        results.push(result)
      } catch (error) {
        console.error(`Error checking npm package ${name}:`, error)
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  private async handlePyProjectVersionCheck(args: any) {
    if (!args.dependencies || typeof args.dependencies !== 'object') {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid dependencies object from pyproject.toml'
      )
    }

    const results: PackageVersion[] = []
    const dependencies = args.dependencies as PyProjectDependencies

    // Process main dependencies
    if (dependencies.dependencies) {
      for (const [name, version] of Object.entries(dependencies.dependencies)) {
        try {
          const result = await this.getPyPiPackageVersion(name, version)
          results.push(result)
        } catch (error) {
          console.error(`Error checking PyPI package ${name}:`, error)
        }
      }
    }

    // Process optional dependencies
    if (dependencies['optional-dependencies']) {
      for (const [group, deps] of Object.entries(dependencies['optional-dependencies'])) {
        for (const [name, version] of Object.entries(deps)) {
          try {
            const result = await this.getPyPiPackageVersion(name, version)
            result.name = `${name} (optional: ${group})`
            results.push(result)
          } catch (error) {
            console.error(`Error checking PyPI package ${name}:`, error)
          }
        }
      }
    }

    // Process dev dependencies
    if (dependencies['dev-dependencies']) {
      for (const [name, version] of Object.entries(dependencies['dev-dependencies'])) {
        try {
          const result = await this.getPyPiPackageVersion(name, version)
          result.name = `${name} (dev)`
          results.push(result)
        } catch (error) {
          console.error(`Error checking PyPI package ${name}:`, error)
        }
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  private async handlePythonVersionCheck(args: any) {
    if (!args.requirements || !Array.isArray(args.requirements)) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid requirements array'
      )
    }

    const results: PackageVersion[] = []
    for (const requirement of args.requirements) {
      if (typeof requirement !== 'string') continue

      // Parse package name and version from requirement string
      const match = requirement.match(/^([a-zA-Z0-9-_.]+)([=<>~!]+.*)?$/)
      if (!match) continue

      const [, name, version = '0.0.0'] = match

      try {
        const result = await this.getPyPiPackageVersion(name, version)
        results.push(result)
      } catch (error) {
        console.error(`Error checking PyPI package ${name}:`, error)
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  private async handleBulkVersionCheck(args: any) {
    if (!args.packages || !Array.isArray(args.packages)) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid packages array'
      )
    }

    const results: PackageVersion[] = []
    for (const pkg of args.packages) {
      if (!pkg || typeof pkg !== 'object' || !pkg.name || !pkg.registry) continue

      try {
        const result = pkg.registry === 'npm'
          ? await this.getNpmPackageVersion(pkg.name, pkg.currentVersion)
          : await this.getPyPiPackageVersion(pkg.name, pkg.currentVersion)
        results.push(result)
      } catch (error) {
        console.error(`Error checking package ${pkg.name}:`, error)
      }
    }

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    }
  }

  async run() {
    const transport = new StdioServerTransport()
    await this.server.connect(transport)
    console.error('Package Version MCP server running on stdio')
  }
}

const server = new PackageVersionServer()
server.run().catch(console.error)
