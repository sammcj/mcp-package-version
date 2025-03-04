import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import {
  PackageHandler,
  DockerImageQuery,
  DockerImageVersion
} from '../types/index.js'

export class DockerHandler implements PackageHandler {
  private dockerHubRegistry = 'https://registry.hub.docker.com/v2'
  private githubContainerRegistry = 'https://ghcr.io/v2'

  private async getDockerHubToken(repository: string): Promise<string> {
    try {
      const response = await axios.get(
        `https://auth.docker.io/token?service=registry.docker.io&scope=repository:${repository}:pull`
      )
      return response.data.token
    } catch (error: any) {
      console.error(`Error getting Docker Hub token for ${repository}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to authenticate with Docker Hub for ${repository}`
      )
    }
  }

  private async getGithubContainerRegistryToken(repository: string): Promise<string | null> {
    // GitHub Container Registry can be accessed anonymously for public images
    // For private images, a token would be needed from environment variables
    const token = process.env.GITHUB_TOKEN
    return token || null
  }

  private async getCustomRegistryToken(registry: string, repository: string): Promise<string | null> {
    // For custom registries, authentication would typically be provided via environment variables
    // This is a simplified implementation
    const username = process.env.CUSTOM_REGISTRY_USERNAME
    const password = process.env.CUSTOM_REGISTRY_PASSWORD
    const token = process.env.CUSTOM_REGISTRY_TOKEN

    if (token) {
      return token
    } else if (username && password) {
      // Some registries support basic auth token generation
      // This is a simplified implementation
      return Buffer.from(`${username}:${password}`).toString('base64')
    }

    return null
  }

  private async fetchTags(
    registry: string,
    repository: string,
    token: string | null,
    limit: number = 10
  ): Promise<{ name: string; tags: string[]; digests?: Record<string, string>; details?: any[] }> {
    try {
      const headers: Record<string, string> = {}
      if (token) {
        headers.Authorization = `Bearer ${token}`
      }

      // First, get the list of tags
      const tagsUrl = `${registry}/${repository}/tags/list`
      const tagsResponse = await axios.get(tagsUrl, { headers })

      const tags = tagsResponse.data.tags || []
      const limitedTags = tags.slice(0, limit)

      // For Docker Hub and registries that support the v2 API, we can get more details
      const details: any[] = []
      const digests: Record<string, string> = {}

      // Get manifest for each tag to extract digest and other details
      for (const tag of limitedTags) {
        try {
          const manifestUrl = `${registry}/${repository}/manifests/${tag}`
          const manifestHeaders = { ...headers, Accept: 'application/vnd.docker.distribution.manifest.v2+json' }
          const manifestResponse = await axios.get(manifestUrl, { headers: manifestHeaders })

          if (manifestResponse.headers['docker-content-digest']) {
            digests[tag] = manifestResponse.headers['docker-content-digest']
          }

          details.push({
            tag,
            digest: manifestResponse.headers['docker-content-digest'],
            contentType: manifestResponse.headers['content-type'],
            size: manifestResponse.data.config?.size,
            layers: manifestResponse.data.layers?.length
          })
        } catch (error: any) {
          console.error(`Error fetching manifest for ${repository}:${tag}:`, error)
        }
      }

      return {
        name: repository,
        tags: limitedTags,
        digests,
        details
      }
    } catch (error: any) {
      console.error(`Error fetching tags for ${repository}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch tags for ${repository}: ${error.message || 'Unknown error'}`
      )
    }
  }

  private async getDockerHubTags(
    image: string,
    limit: number = 10,
    includeDigest: boolean = false
  ): Promise<DockerImageVersion[]> {
    // For Docker Hub, the repository format is typically namespace/image
    const repository = image.includes('/') ? image : `library/${image}`
    const token = await this.getDockerHubToken(repository)

    const result = await this.fetchTags(
      this.dockerHubRegistry,
      repository,
      token,
      limit
    )

    // For Docker Hub, we can also get additional information about each tag
    const versions: DockerImageVersion[] = []

    for (const tag of result.tags) {
      try {
        // Get more details about this specific tag
        const detailsUrl = `https://hub.docker.com/v2/repositories/${repository}/tags/${tag}`
        const detailsResponse = await axios.get(detailsUrl)
        const tagInfo = detailsResponse.data

        versions.push({
          name: image,
          tag,
          registry: 'docker',
          digest: includeDigest ? result.digests?.[tag] : undefined,
          created: tagInfo.last_updated,
          size: tagInfo.full_size ? `${Math.round(tagInfo.full_size / 1024 / 1024)} MB` : undefined
        })
      } catch (error: any) {
        // If we can't get detailed info, just add the basic tag info
        versions.push({
          name: image,
          tag,
          registry: 'docker',
          digest: includeDigest ? result.digests?.[tag] : undefined
        })
      }
    }

    return versions
  }

  private async getGithubContainerRegistryTags(
    image: string,
    limit: number = 10,
    includeDigest: boolean = false
  ): Promise<DockerImageVersion[]> {
    const token = await this.getGithubContainerRegistryToken(image)

    const result = await this.fetchTags(
      this.githubContainerRegistry,
      image,
      token,
      limit
    )

    return result.tags.map(tag => ({
      name: image,
      tag,
      registry: 'ghcr',
      digest: includeDigest ? result.digests?.[tag] : undefined
    }))
  }

  private async getCustomRegistryTags(
    registry: string,
    image: string,
    limit: number = 10,
    includeDigest: boolean = false
  ): Promise<DockerImageVersion[]> {
    const token = await this.getCustomRegistryToken(registry, image)

    // Remove protocol and trailing slash from registry URL for API calls
    const registryUrl = registry.replace(/^(https?:\/\/)/, '').replace(/\/$/, '')
    const apiUrl = `https://${registryUrl}/v2`

    const result = await this.fetchTags(
      apiUrl,
      image,
      token,
      limit
    )

    return result.tags.map(tag => ({
      name: image,
      tag,
      registry: 'custom',
      digest: includeDigest ? result.digests?.[tag] : undefined
    }))
  }

  async getLatestVersion(args: DockerImageQuery) {
    if (!args.image) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Image name is required'
      )
    }

    const limit = args.limit || 10
    const includeDigest = args.includeDigest || false
    let results: DockerImageVersion[] = []

    try {
      switch (args.registry) {
        case 'ghcr':
          results = await this.getGithubContainerRegistryTags(args.image, limit, includeDigest)
          break
        case 'custom':
          if (!args.customRegistry) {
            throw new McpError(
              ErrorCode.InvalidParams,
              'Custom registry URL is required when registry type is "custom"'
            )
          }
          results = await this.getCustomRegistryTags(args.customRegistry, args.image, limit, includeDigest)
          break
        case 'dockerhub':
        default:
          results = await this.getDockerHubTags(args.image, limit, includeDigest)
          break
      }

      // Filter tags if filterTags is provided
      if (args.filterTags && args.filterTags.length > 0) {
        const filterRegexes = args.filterTags.map(pattern => new RegExp(pattern))
        results = results.filter(result =>
          filterRegexes.some(regex => regex.test(result.tag))
        )
      }

      return {
        content: [
          {
            type: 'text',
            text: JSON.stringify(results, null, 2),
          },
        ],
      }
    } catch (error: any) {
      console.error(`Error checking Docker image ${args.image}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch Docker image information: ${error.message || 'Unknown error'}`
      )
    }
  }
}
