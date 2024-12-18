import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import { PackageVersion, PyProjectDependencies, PackageHandler } from '../types/index.js'

export class PythonHandler implements PackageHandler {
  private registry = 'https://pypi.org/pypi'

  private async getPackageVersion(
    packageName: string,
    currentVersion?: string,
    label?: string
  ): Promise<PackageVersion> {
    try {
      const response = await axios.get(
        `${this.registry}/${encodeURIComponent(packageName)}/json`
      )

      const latestVersion = response.data.info.version
      if (!latestVersion) {
        throw new Error('Latest version not found')
      }

      const result: PackageVersion = {
        name: label ? `${packageName} (${label})` : packageName,
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

  async getLatestVersionFromRequirements(args: { requirements: string[] }) {
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
        const result = await this.getPackageVersion(name, version)
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

  async getLatestVersion(args: { dependencies: PyProjectDependencies }) {
    if (!args.dependencies || typeof args.dependencies !== 'object') {
      throw new McpError(
        ErrorCode.InvalidParams,
        'Invalid dependencies object from pyproject.toml'
      )
    }

    const results: PackageVersion[] = []
    const dependencies = args.dependencies

    // Process main dependencies
    if (dependencies.dependencies) {
      for (const [name, version] of Object.entries(dependencies.dependencies)) {
        try {
          const result = await this.getPackageVersion(name, version)
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
            const result = await this.getPackageVersion(name, version, `optional: ${group}`)
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
          const result = await this.getPackageVersion(name, version, 'dev')
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
}
