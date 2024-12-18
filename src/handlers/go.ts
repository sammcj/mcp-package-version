import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import { PackageVersion, GoModule, PackageHandler } from '../types/index.js'

export class GoHandler implements PackageHandler {
  private proxyBase = 'https://proxy.golang.org'

  private async getPackageVersion(
    path: string,
    currentVersion?: string
  ): Promise<PackageVersion> {
    try {
      // First get the list of versions
      const response = await axios.get(`${this.proxyBase}/${path}/@v/list`)
      const versions = response.data.trim().split('\n')

      if (!versions.length) {
        throw new Error('No versions found')
      }

      // Get the latest version (last in the list)
      const latestVersion = versions[versions.length - 1]

      const result: PackageVersion = {
        name: path,
        latestVersion,
        registry: 'go',
      }

      if (currentVersion) {
        // Remove any 'v' prefix from the current version
        const cleanCurrentVersion = currentVersion.replace(/^v/, '')
        result.currentVersion = cleanCurrentVersion
      }

      return result
    } catch (error) {
      console.error(`Error fetching Go package ${path}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch Go package ${path}`
      )
    }
  }

  async getLatestVersion(args: { dependencies: GoModule }) {
    if (!args.dependencies || !args.dependencies.require) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid Go module dependencies'
      )
    }

    const results: PackageVersion[] = []
    const { require = [], replace = [] } = args.dependencies

    // Process required dependencies
    for (const dep of require) {
      if (!dep.path) continue

      try {
        const result = await this.getPackageVersion(dep.path, dep.version)
        results.push(result)
      } catch (error) {
        console.error(`Error checking Go package ${dep.path}:`, error)
      }
    }

    // Process replaced dependencies
    for (const rep of replace) {
      if (!rep.old || !rep.new) continue

      try {
        const result = await this.getPackageVersion(rep.new, rep.version)
        result.name = `${rep.new} (replaces ${rep.old})`
        results.push(result)
      } catch (error) {
        console.error(`Error checking Go package ${rep.new}:`, error)
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
