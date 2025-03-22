import axios from 'axios'
import { McpError, ErrorCode } from '@modelcontextprotocol/sdk/types.js'
import {
  PackageHandler,
  GitHubActionQuery,
  GitHubActionVersion,
  GitHubActionInput
} from '../types/index.js'

export class GitHubActionsHandler implements PackageHandler {
  private githubApiUrl = 'https://api.github.com'
  private githubToken = process.env.GITHUB_TOKEN || ''

  private getHeaders() {
    const headers: Record<string, string> = {
      'Accept': 'application/vnd.github.v3+json'
    }

    if (this.githubToken) {
      headers['Authorization'] = `token ${this.githubToken}`
    }

    return headers
  }

  private async getLatestActionVersion(action: GitHubActionInput): Promise<GitHubActionVersion> {
    try {
      const { owner, repo, currentVersion } = action
      const headers = this.getHeaders()

      // First, try to get releases
      const releasesUrl = `${this.githubApiUrl}/repos/${owner}/${repo}/releases`
      const releasesResponse = await axios.get(releasesUrl, { headers })

      if (releasesResponse.data && releasesResponse.data.length > 0) {
        // Sort releases by published date (newest first)
        const releases = releasesResponse.data.sort((a: any, b: any) =>
          new Date(b.published_at).getTime() - new Date(a.published_at).getTime()
        )

        // Get the latest release
        const latestRelease = releases[0]

        // Extract version from tag name (remove 'v' prefix if present)
        const latestVersion = latestRelease.tag_name.replace(/^v/, '')

        // Parse version components
        const versionParts = latestVersion.split('.')
        const latestMajorVersion = versionParts[0] ? `${versionParts[0]}` : undefined
        const latestMinorVersion = versionParts[1] ? `${versionParts[0]}.${versionParts[1]}` : undefined
        const latestPatchVersion = latestVersion

        return {
          name: `${owner}/${repo}`,
          owner,
          repo,
          currentVersion,
          latestVersion,
          latestMajorVersion,
          latestMinorVersion,
          latestPatchVersion,
          publishedAt: latestRelease.published_at,
          url: latestRelease.html_url
        }
      }

      // If no releases, try to get tags
      const tagsUrl = `${this.githubApiUrl}/repos/${owner}/${repo}/tags`
      const tagsResponse = await axios.get(tagsUrl, { headers })

      if (tagsResponse.data && tagsResponse.data.length > 0) {
        // Get the first tag (usually the latest)
        const latestTag = tagsResponse.data[0]

        // Extract version from tag name (remove 'v' prefix if present)
        const latestVersion = latestTag.name.replace(/^v/, '')

        // Parse version components
        const versionParts = latestVersion.split('.')
        const latestMajorVersion = versionParts[0] ? `${versionParts[0]}` : undefined
        const latestMinorVersion = versionParts[1] ? `${versionParts[0]}.${versionParts[1]}` : undefined
        const latestPatchVersion = latestVersion

        return {
          name: `${owner}/${repo}`,
          owner,
          repo,
          currentVersion,
          latestVersion,
          latestMajorVersion,
          latestMinorVersion,
          latestPatchVersion,
          url: latestTag.commit.url
        }
      }

      // If no releases or tags, check if the repo exists
      const repoUrl = `${this.githubApiUrl}/repos/${owner}/${repo}`
      await axios.get(repoUrl, { headers })

      // If we get here, the repo exists but has no releases or tags
      return {
        name: `${owner}/${repo}`,
        owner,
        repo,
        currentVersion,
        latestVersion: 'unknown',
        url: `https://github.com/${owner}/${repo}`
      }
    } catch (error: any) {
      console.error(`Error fetching GitHub Action ${action.owner}/${action.repo}:`, error)

      // Check if it's a 404 error (repo not found)
      if (error.response && error.response.status === 404) {
        return {
          name: `${action.owner}/${action.repo}`,
          owner: action.owner,
          repo: action.repo,
          currentVersion: action.currentVersion,
          latestVersion: 'not found',
          skipped: true,
          skipReason: 'Repository not found'
        } as GitHubActionVersion
      }

      // Check if it's a rate limit error
      if (error.response && error.response.status === 403 && error.response.headers['x-ratelimit-remaining'] === '0') {
        return {
          name: `${action.owner}/${action.repo}`,
          owner: action.owner,
          repo: action.repo,
          currentVersion: action.currentVersion,
          latestVersion: 'unknown',
          skipped: true,
          skipReason: 'GitHub API rate limit exceeded'
        } as GitHubActionVersion
      }

      // Other errors
      return {
        name: `${action.owner}/${action.repo}`,
        owner: action.owner,
        repo: action.repo,
        currentVersion: action.currentVersion,
        latestVersion: 'error',
        skipped: true,
        skipReason: `Error: ${error.message || 'Unknown error'}`
      } as GitHubActionVersion
    }
  }

  async getLatestVersion(args: GitHubActionQuery) {
    if (!args.actions || !Array.isArray(args.actions) || args.actions.length === 0) {
      throw new McpError(
        ErrorCode.InvalidParams,
        'At least one GitHub Action must be provided'
      )
    }

    try {
      const results: GitHubActionVersion[] = []

      // Process each action in parallel
      const promises = args.actions.map(action => this.getLatestActionVersion(action))
      const actionVersions = await Promise.all(promises)

      results.push(...actionVersions)

      // Filter out unnecessary details if includeDetails is false
      if (args.includeDetails !== true) {
        results.forEach(result => {
          delete result.publishedAt
          delete result.url
        })
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
      console.error('Error checking GitHub Actions:', error)
      throw new McpError(
        ErrorCode.InternalError,
        `Failed to fetch GitHub Actions information: ${error.message || 'Unknown error'}`
      )
    }
  }
}
