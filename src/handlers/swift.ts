import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import {
  PackageVersion,
  SwiftDependency,
  PackageHandler,
  VersionConstraint,
  VersionConstraints
} from '../types/index.js'

export class SwiftHandler implements PackageHandler {
  private async getPackageVersion(
    packageUrl: string,
    currentVersion?: string,
    versionRequirement?: string,
    constraints?: VersionConstraint
  ): Promise<PackageVersion> {
    // Extract package name from URL
    const packageName = packageUrl.split('/').pop()?.replace('.git', '') || packageUrl

    // Check if package should be excluded
    if (constraints?.excludePackage) {
      return {
        name: packageName,
        currentVersion: currentVersion,
        latestVersion: currentVersion || 'unknown',
        registry: 'swift',
        skipped: true,
        skipReason: 'Package excluded from updates'
      }
    }

    try {
      // For GitHub repositories, we can use the GitHub API to get the latest release
      if (packageUrl.includes('github.com')) {
        // Convert GitHub URL to API URL
        // Example: https://github.com/apple/swift-argument-parser -> https://api.github.com/repos/apple/swift-argument-parser/releases
        const apiUrl = packageUrl
          .replace('https://github.com/', 'https://api.github.com/repos/')
          .replace('.git', '') + '/releases'

        const response = await axios.get(apiUrl)

        if (!response.data || response.data.length === 0) {
          throw new Error('No releases found')
        }

        // Find the latest release that's not a pre-release
        let latestRelease = response.data.find((release: any) => !release.prerelease)

        // If no stable release is found, use the latest release
        if (!latestRelease) {
          latestRelease = response.data[0]
        }

        let latestVersion = latestRelease.tag_name

        // Remove 'v' prefix if present
        if (latestVersion.startsWith('v')) {
          latestVersion = latestVersion.substring(1)
        }

        // If major version constraint exists, check if the latest version complies
        if (constraints?.majorVersion !== undefined) {
          const major = parseInt(latestVersion.split('.')[0])
          if (major !== constraints.majorVersion) {
            // Find the latest release within the specified major version
            const constrainedRelease = response.data.find((release: any) => {
              let version = release.tag_name
              if (version.startsWith('v')) {
                version = version.substring(1)
              }
              return parseInt(version.split('.')[0]) === constraints.majorVersion
            })

            if (constrainedRelease) {
              latestVersion = constrainedRelease.tag_name
              if (latestVersion.startsWith('v')) {
                latestVersion = latestVersion.substring(1)
              }
            }
          }
        }

        const result: PackageVersion = {
          name: packageName,
          latestVersion,
          registry: 'swift'
        }

        if (currentVersion) {
          result.currentVersion = currentVersion
        }

        if (constraints?.majorVersion !== undefined) {
          result.skipReason = `Limited to major version ${constraints.majorVersion}`
        }

        return result
      } else {
        // For non-GitHub repositories, we can't easily determine the latest version
        // Return the current version as the latest version
        return {
          name: packageName,
          currentVersion: currentVersion,
          latestVersion: currentVersion || 'unknown',
          registry: 'swift',
          skipped: true,
          skipReason: 'Non-GitHub repository, cannot determine latest version'
        }
      }
    } catch (error) {
      console.error(`Error fetching Swift package ${packageName}:`, error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch Swift package ${packageName}`
      )
    }
  }

  async getLatestVersion(args: {
    dependencies: SwiftDependency[],
    constraints?: VersionConstraints
  }) {
    if (!args.dependencies || !Array.isArray(args.dependencies)) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid dependencies array'
      )
    }

    const results: PackageVersion[] = []
    for (const dependency of args.dependencies) {
      try {
        const result = await this.getPackageVersion(
          dependency.url,
          dependency.version,
          dependency.requirement,
          args.constraints?.[dependency.url]
        )
        results.push(result)
      } catch (error) {
        console.error(`Error checking Swift package ${dependency.url}:`, error)
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
