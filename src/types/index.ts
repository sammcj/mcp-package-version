export * from './bedrock.js';
export * from './docker.js';
export * from './github-actions.js';

export interface PackageVersion {
  name: string
  currentVersion?: string
  latestVersion: string
  registry: 'npm' | 'pypi' | 'maven' | 'go' | 'docker' | 'ghcr' | 'custom' | 'swift' | 'github-actions'
  skipped?: boolean
  skipReason?: string
}

export interface VersionConstraint {
  majorVersion?: number
  excludePackage?: boolean
}

export interface VersionConstraints {
  [packageName: string]: VersionConstraint
}

export interface RegistryConfig {
  url: string
}

// JavaScript/Node.js types
export interface NpmDependencies {
  [key: string]: string
}

// Python types
export interface PyProjectDependencies {
  dependencies?: { [key: string]: string }
  'optional-dependencies'?: { [key: string]: { [key: string]: string } }
  'dev-dependencies'?: { [key: string]: string }
}

// Java types
export interface MavenDependency {
  groupId: string
  artifactId: string
  version?: string
  scope?: string
}

export interface GradleDependency {
  configuration: string
  group: string
  name: string
  version?: string
}

// Go types
export interface GoModule {
  module: string
  require?: GoRequire[]
  replace?: GoReplace[]
}

export interface GoRequire {
  path: string
  version: string
}

export interface GoReplace {
  old: string
  new: string
  version?: string
}

// Swift types
export interface SwiftDependency {
  url: string
  version?: string
  requirement?: string // e.g., 'from', 'upToNextMajor', 'exact'
}

// Common handler interface
export interface PackageHandler {
  getLatestVersion(args: any): Promise<{ content: { type: string; text: string }[] }>
}
