import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import { PackageVersion, NpmDependencies, PackageHandler } from '../types/index.js'

export class NpmHandler implements PackageHandler {
  private registry = 'https://registry.npmjs.org'

  private async getPackageVersion(
    packageName: string,
    currentVersion?: string
  ): Promise<PackageVersion> {
    try {
      const response = await axios.get(
        `${this.registry}/${encodeURIComponent(packageName)}`
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

  async getLatestVersion(args: { dependencies: NpmDependencies }) {
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
        const result = await this.getPackageVersion(name, version)
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
}
