export interface DockerImageVersion {
  name: string;
  tag: string;
  registry: string;
  digest?: string;
  created?: string;
  size?: string;
}

export interface DockerRegistryConfig {
  url: string;
  username?: string;
  password?: string;
  token?: string;
}

export interface DockerImageQuery {
  image: string;
  registry?: 'dockerhub' | 'ghcr' | 'custom';
  customRegistry?: string;
  limit?: number;
  filterTags?: string[];
  includeDigest?: boolean;
}
