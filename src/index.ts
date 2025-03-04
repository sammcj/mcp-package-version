#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ErrorCode,
  ListToolsRequestSchema,
  McpError,
} from '@modelcontextprotocol/sdk/types.js'

import {
  NpmDependencies,
  PyProjectDependencies,
  MavenDependency,
  GradleDependency,
  GoModule,
  VersionConstraints,
  DockerImageQuery,
} from './types/index.js'
import { NpmHandler } from './handlers/npm.js'
import { PythonHandler } from './handlers/python.js'
import { JavaHandler } from './handlers/java.js'
import { GoHandler } from './handlers/go.js'
import { BedrockHandler } from './handlers/bedrock.js'
import { DockerHandler } from './handlers/docker.js'

class PackageVersionServer {
  private server: Server
  private npmHandler: NpmHandler
  private pythonHandler: PythonHandler
  private javaHandler: JavaHandler
  private goHandler: GoHandler
  private bedrockHandler: BedrockHandler
  private dockerHandler: DockerHandler

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

    this.npmHandler = new NpmHandler()
    this.pythonHandler = new PythonHandler()
    this.javaHandler = new JavaHandler()
    this.goHandler = new GoHandler()
    this.bedrockHandler = new BedrockHandler()
    this.dockerHandler = new DockerHandler()

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
              constraints: {
                type: 'object',
                additionalProperties: {
                  type: 'object',
                  properties: {
                    majorVersion: {
                      type: 'number',
                      description: 'Limit updates to this major version',
                    },
                    excludePackage: {
                      type: 'boolean',
                      description: 'Exclude this package from updates',
                    },
                  },
                },
                description: 'Optional constraints for specific packages',
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
          name: 'check_go_versions',
          description: 'Check latest stable versions for Go packages in go.mod',
          inputSchema: {
            type: 'object',
            properties: {
              dependencies: {
                type: 'object',
                properties: {
                  module: {
                    type: 'string',
                    description: 'Module name',
                  },
                  require: {
                    type: 'array',
                    items: {
                      type: 'object',
                      properties: {
                        path: {
                          type: 'string',
                          description: 'Package import path',
                        },
                        version: {
                          type: 'string',
                          description: 'Current version',
                        },
                      },
                      required: ['path'],
                    },
                    description: 'Required dependencies',
                  },
                  replace: {
                    type: 'array',
                    items: {
                      type: 'object',
                      properties: {
                        old: {
                          type: 'string',
                          description: 'Original package path',
                        },
                        new: {
                          type: 'string',
                          description: 'Replacement package path',
                        },
                        version: {
                          type: 'string',
                          description: 'Current version',
                        },
                      },
                      required: ['old', 'new'],
                    },
                    description: 'Replacement dependencies',
                  },
                },
                required: ['module'],
                description: 'Dependencies from go.mod',
              },
            },
            required: ['dependencies'],
          },
        },
        {
          name: 'check_bedrock_models',
          description: 'Search, list, and get information about Amazon Bedrock models',
          inputSchema: {
            type: 'object',
            properties: {
              action: {
                type: 'string',
                enum: ['list', 'search', 'get'],
                description: 'Action to perform: list all models, search for models, or get a specific model',
                default: 'list'
              },
              query: {
                type: 'string',
                description: 'Search query for model name or ID (used with action: "search")'
              },
              provider: {
                type: 'string',
                description: 'Filter by provider name (used with action: "search")'
              },
              region: {
                type: 'string',
                description: 'Filter by AWS region (used with action: "search")'
              },
              modelId: {
                type: 'string',
                description: 'Model ID to retrieve (used with action: "get")'
              }
            }
          }
        },
        {
          name: 'get_latest_bedrock_model',
          description: 'Get the latest Claude Sonnet model from Amazon Bedrock (best for coding tasks)',
          inputSchema: {
            type: 'object',
            properties: {}
          }
        },
        {
          name: 'check_docker_tags',
          description: 'Check available tags for Docker container images from Docker Hub, GitHub Container Registry, or custom registries',
          inputSchema: {
            type: 'object',
            properties: {
              image: {
                type: 'string',
                description: 'Docker image name (e.g., "nginx", "ubuntu", "ghcr.io/owner/repo")'
              },
              registry: {
                type: 'string',
                enum: ['dockerhub', 'ghcr', 'custom'],
                description: 'Registry to check (dockerhub, ghcr, or custom)',
                default: 'dockerhub'
              },
              customRegistry: {
                type: 'string',
                description: 'URL for custom registry (required when registry is "custom")'
              },
              limit: {
                type: 'number',
                description: 'Maximum number of tags to return',
                default: 10
              },
              filterTags: {
                type: 'array',
                items: {
                  type: 'string'
                },
                description: 'Array of regex patterns to filter tags'
              },
              includeDigest: {
                type: 'boolean',
                description: 'Include image digest in results',
                default: false
              }
            },
            required: ['image']
          }
        },
      ],
    }))

    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      if (!request.params.arguments) {
        throw new McpError(
          ErrorCode.InvalidParams,
          'Missing arguments'
        )
      }

      switch (request.params.name) {
        case 'check_npm_versions':
          return this.npmHandler.getLatestVersion(request.params.arguments as {
            dependencies: NpmDependencies,
            constraints?: VersionConstraints
          })
        case 'check_python_versions':
          return this.pythonHandler.getLatestVersionFromRequirements(request.params.arguments as { requirements: string[] })
        case 'check_pyproject_versions':
          return this.pythonHandler.getLatestVersion(request.params.arguments as { dependencies: PyProjectDependencies })
        case 'check_maven_versions':
          return this.javaHandler.getLatestVersionFromMaven(request.params.arguments as { dependencies: MavenDependency[] })
        case 'check_gradle_versions':
          return this.javaHandler.getLatestVersion(request.params.arguments as { dependencies: GradleDependency[] })
        case 'check_go_versions':
          return this.goHandler.getLatestVersion(request.params.arguments as { dependencies: GoModule })
        case 'check_bedrock_models':
          return this.bedrockHandler.getLatestVersion(request.params.arguments)
        case 'get_latest_bedrock_model':
          // Set the action to get_latest_claude_sonnet to use the specialized method
          return this.bedrockHandler.getLatestVersion({ action: 'get_latest_claude_sonnet' })
        case 'check_docker_tags':
          return this.dockerHandler.getLatestVersion(request.params.arguments as any)
        default:
          throw new McpError(
            ErrorCode.MethodNotFound,
            `Unknown tool: ${request.params.name}`
          )
      }
    })
  }

  async run() {
    const transport = new StdioServerTransport()
    await this.server.connect(transport)
    console.error('Package Version MCP server running on stdio')
  }
}

const server = new PackageVersionServer()
server.run().catch(console.error)
