import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import {
  PackageVersion,
  NpmDependencies,
  PackageHandler,
  VersionConstraint,
  VersionConstraints
} from '../types/index.js'

export class NpmHandler implements PackageHandler {
  private registry = 'https://registry.npmjs.org'

  private async getPackageVersion(
    packageName: string,
    currentVersion?: string,
    constraints?: VersionConstraint
  ): Promise<PackageVersion> {
    // Check if package should be excluded
    if (constraints?.excludePackage) {
      return {
        name: packageName,
        currentVersion: currentVersion?.replace(/^[\^~]/, ''),
        latestVersion: currentVersion?.replace(/^[\^~]/, '') || 'unknown',
        registry: 'npm',
        skipped: true,
        skipReason: 'Package excluded from updates'
      }
    }
    try {
      const response = await axios.get(
        `${this.registry}/${encodeURIComponent(packageName)}`
      )

      let latestVersion = response.data['dist-tags']?.latest
      if (!latestVersion) {
        throw new Error('Latest version not found')
      }

      // If major version constraint exists, find the latest version within that major
      if (constraints?.majorVersion !== undefined) {
        const versions = Object.keys(response.data.versions || {})
        const constrainedVersions = versions.filter(v => {
          const major = parseInt(v.split('.')[0])
          return major === constraints.majorVersion
        })

        if (constrainedVersions.length > 0) {
          // Sort versions and get the latest within the major version
          latestVersion = constrainedVersions.sort((a, b) => {
            const [aMajor, aMinor, aPatch] = a.split('.').map(Number)
            const [bMajor, bMinor, bPatch] = b.split('.').map(Number)
            if (aMajor !== bMajor) return bMajor - aMajor
            if (aMinor !== bMinor) return bMinor - aMinor
            return bPatch - aPatch
          })[0]
        }
      }

      const result: PackageVersion = {
        name: packageName,
        latestVersion,
        registry: 'npm'
      }

      if (currentVersion) {
        // Remove any leading ^ or ~ from the current version
        const cleanCurrentVersion = currentVersion.replace(/^[\^~]/, '')
        result.currentVersion = cleanCurrentVersion
      }

      if (constraints?.majorVersion !== undefined) {
        result.skipReason = `Limited to major version ${constraints.majorVersion}`
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

  async getLatestVersion(args: {
    dependencies: NpmDependencies,
    constraints?: VersionConstraints
  }) {
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
        const result = await this.getPackageVersion(
          name,
          version,
          args.constraints?.[name]
        )
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
