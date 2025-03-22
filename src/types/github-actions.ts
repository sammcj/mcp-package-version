export interface GitHubActionVersion {
  name: string;
  owner: string;
  repo: string;
  currentVersion?: string;
  latestVersion: string;
  latestMajorVersion?: string;
  latestMinorVersion?: string;
  latestPatchVersion?: string;
  publishedAt?: string;
  url?: string;
}

export interface GitHubActionQuery {
  actions: GitHubActionInput[];
  includeDetails?: boolean;
}

export interface GitHubActionInput {
  owner: string;
  repo: string;
  currentVersion?: string;
}
