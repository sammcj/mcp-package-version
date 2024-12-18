import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import { PackageVersion, MavenDependency, GradleDependency, PackageHandler } from '../types/index.js'

export class JavaHandler implements PackageHandler {
  private registry = 'https://search.maven.org/solrsearch/select'

  private async getPackageVersion(
    groupId: string,
    artifactId: string,
    currentVersion?: string,
    scope?: string
  ): Promise<PackageVersion> {
    try {
      const query = `g:"${groupId}" AND a:"${artifactId}"`
      const response = await axios.get(this.registry, {
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

  async getLatestVersionFromMaven(args: { dependencies: MavenDependency[] }) {
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
        const result = await this.getPackageVersion(
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

  async getLatestVersion(args: { dependencies: GradleDependency[] }) {
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
        const result = await this.getPackageVersion(
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
}
