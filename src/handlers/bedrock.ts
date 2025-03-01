import axios from 'axios';
import { BedrockModel, BedrockModelSearchResult } from '../types/bedrock.js';
import { PackageHandler } from '../types/index.js';

export class BedrockHandler implements PackageHandler {
  private modelsCache: BedrockModel[] | null = null;
  private lastFetchTime: number = 0;
  private readonly cacheTTL = 3600000; // 1 hour in milliseconds
  private readonly docsUrl = 'https://docs.aws.amazon.com/bedrock/latest/userguide/models-supported.html';

  constructor() {}

  /**
   * Fetches the latest Bedrock model information from AWS documentation
   */
  private async fetchModels(): Promise<BedrockModel[]> {
    // Check if we have a valid cache
    const now = Date.now();
    if (this.modelsCache && (now - this.lastFetchTime < this.cacheTTL)) {
      return this.modelsCache;
    }

    try {
      const response = await axios.get(this.docsUrl);
      const html = response.data;

      // Parse the HTML to extract the model information from the first table
      const models = this.parseModelsFromHtml(html);

      // Update cache
      this.modelsCache = models;
      this.lastFetchTime = now;

      return models;
    } catch (error) {
      console.error('Error fetching Bedrock models:', error);
      // If we have a cache, return it even if it's expired
      if (this.modelsCache) {
        return this.modelsCache;
      }
      throw error;
    }
  }

  /**
   * Parses the HTML to extract model information from the first table
   */
  private parseModelsFromHtml(html: string): BedrockModel[] {
    const models: BedrockModel[] = [];

    try {
      // Find the first table in the HTML
      // AWS docs typically use more complex table structure with classes
      const tableRegex = /<table[\s\S]*?>[\s\S]*?<\/table>/i;
      const tableMatch = html.match(tableRegex);

      if (!tableMatch) {
        console.error('No table found in HTML');
        return models;
      }

      const tableHtml = tableMatch[0];

      // Extract rows from the table (skip the header row)
      const rowRegex = /<tr[\s\S]*?>[\s\S]*?<\/tr>/gi;
      const rows = tableHtml.match(rowRegex);

      if (!rows || rows.length <= 1) {
        console.error('No rows found in table');
        return models;
      }

      // Skip the header row
      for (let i = 1; i < rows.length; i++) {
        const row = rows[i];

        // Extract cells from the row - AWS docs might use th or td
        const cellRegex = /<t[dh][\s\S]*?>[\s\S]*?<\/t[dh]>/gi;
        const cells = row.match(cellRegex);

        if (!cells || cells.length < 7) {
          console.error(`Invalid row format (cells: ${cells?.length}):`, row.substring(0, 100) + '...');
          continue;
        }

        // Extract text from cells
        const provider = this.extractTextFromCell(cells[0]);
        const modelName = this.extractTextFromCell(cells[1]);
        const modelId = this.extractTextFromCell(cells[2]);
        const regionsSupported = this.extractRegionsFromCell(cells[3]);
        const inputModalities = this.extractListFromCell(cells[4]);
        const outputModalities = this.extractListFromCell(cells[5]);
        const streamingSupported = this.extractTextFromCell(cells[6]).toLowerCase() === 'yes';

        // Only add if we have valid data
        if (modelName && modelId) {
          models.push({
            provider,
            modelName,
            modelId,
            regionsSupported,
            inputModalities,
            outputModalities,
            streamingSupported
          });
        }
      }

      console.log(`Successfully parsed ${models.length} models from HTML`);
    } catch (error) {
      console.error('Error parsing HTML:', error);
    }

    return models;
  }

  /**
   * Extracts text from a table cell
   */
  private extractTextFromCell(cell: string): string {
    // Remove HTML tags and trim whitespace
    return cell.replace(/<[^>]*>/g, '').trim();
  }

  /**
   * Extracts a list of regions from a cell
   */
  private extractRegionsFromCell(cell: string): string[] {
    const text = this.extractTextFromCell(cell);
    // Split by whitespace and filter out empty strings
    return text.split(/\s+/).filter(region => region.length > 0);
  }

  /**
   * Extracts a list from a cell (comma-separated values)
   */
  private extractListFromCell(cell: string): string[] {
    const text = this.extractTextFromCell(cell);
    // Split by comma and trim whitespace
    return text.split(',').map(item => item.trim()).filter(item => item.length > 0);
  }

  /**
   * Searches for Bedrock models based on query parameters
   */
  public async searchModels(query: string, provider?: string, region?: string): Promise<BedrockModelSearchResult> {
    const models = await this.fetchModels();

    // Normalize the query for case-insensitive search
    const normalizedQuery = query.toLowerCase().trim();
    const normalizedProvider = provider?.toLowerCase().trim();
    const normalizedRegion = region?.toLowerCase().trim();

    // Filter models based on query parameters
    const filteredModels = models.filter(model => {
      // Match query against model name or model ID (fuzzy search)
      const matchesQuery = !normalizedQuery ||
        this.fuzzyMatch(model.modelName.toLowerCase(), normalizedQuery) ||
        this.fuzzyMatch(model.modelId.toLowerCase(), normalizedQuery);

      // Match provider if specified (fuzzy search)
      const matchesProvider = !normalizedProvider ||
        this.fuzzyMatch(model.provider.toLowerCase(), normalizedProvider);

      // Match region if specified
      const matchesRegion = !normalizedRegion ||
        model.regionsSupported.some(r =>
          r.toLowerCase().includes(normalizedRegion) ||
          normalizedRegion.includes(r.toLowerCase())
        );

      return matchesQuery && matchesProvider && matchesRegion;
    });

    // Sort results by relevance if there's a query
    let sortedModels = filteredModels;
    if (normalizedQuery) {
      sortedModels = filteredModels.sort((a, b) => {
        // Exact matches first
        const aExactMatch =
          a.modelName.toLowerCase() === normalizedQuery ||
          a.modelId.toLowerCase() === normalizedQuery;
        const bExactMatch =
          b.modelName.toLowerCase() === normalizedQuery ||
          b.modelId.toLowerCase() === normalizedQuery;

        if (aExactMatch && !bExactMatch) return -1;
        if (!aExactMatch && bExactMatch) return 1;

        // Then sort by how early the match appears
        const aNameIndex = a.modelName.toLowerCase().indexOf(normalizedQuery);
        const bNameIndex = b.modelName.toLowerCase().indexOf(normalizedQuery);
        const aIdIndex = a.modelId.toLowerCase().indexOf(normalizedQuery);
        const bIdIndex = b.modelId.toLowerCase().indexOf(normalizedQuery);

        const aIndex = aNameIndex >= 0 ? aNameIndex : (aIdIndex >= 0 ? aIdIndex : Number.MAX_SAFE_INTEGER);
        const bIndex = bNameIndex >= 0 ? bNameIndex : (bIdIndex >= 0 ? bIdIndex : Number.MAX_SAFE_INTEGER);

        return aIndex - bIndex;
      });
    }

    return {
      models: sortedModels,
      totalCount: sortedModels.length
    };
  }

  /**
   * Performs a fuzzy match between a string and a query
   */
  private fuzzyMatch(str: string, query: string): boolean {
    // Simple implementation: check if all characters in query appear in order in str
    if (!query) return true;
    if (!str) return false;

    // Direct substring match
    if (str.includes(query)) return true;

    // Check for character-by-character fuzzy match
    let strIndex = 0;
    let queryIndex = 0;

    while (strIndex < str.length && queryIndex < query.length) {
      if (str[strIndex] === query[queryIndex]) {
        queryIndex++;
      }
      strIndex++;
    }

    return queryIndex === query.length;
  }

  /**
   * Lists all available Bedrock models
   */
  public async listModels(): Promise<BedrockModelSearchResult> {
    const models = await this.fetchModels();

    return {
      models,
      totalCount: models.length
    };
  }

  /**
   * Gets a specific Bedrock model by ID
   */
  public async getModelById(modelId: string): Promise<BedrockModel | null> {
    const models = await this.fetchModels();

    return models.find(model => model.modelId === modelId) || null;
  }

  /**
   * Gets the latest Claude Sonnet model
   * This returns the most recent version of Claude Sonnet, which is typically
   * the best model for coding tasks
   */
  public async getLatestClaudeSonnetModel(): Promise<BedrockModel | null> {
    const models = await this.fetchModels();

    // Filter for Claude Sonnet models
    const sonnetModels = models.filter(model =>
      model.provider.toLowerCase().includes('anthropic') &&
      model.modelName.toLowerCase().includes('claude') &&
      model.modelName.toLowerCase().includes('sonnet')
    );

    if (sonnetModels.length === 0) {
      return null;
    }

    // Sort by version number to find the latest
    // Extract version numbers and sort
    const sortedModels = sonnetModels.sort((a, b) => {
      // Extract version numbers (e.g., 3.5, 3.7)
      const aVersionMatch = a.modelName.match(/(\d+\.\d+)/);
      const bVersionMatch = b.modelName.match(/(\d+\.\d+)/);

      const aVersion = aVersionMatch ? parseFloat(aVersionMatch[1]) : 0;
      const bVersion = bVersionMatch ? parseFloat(bVersionMatch[1]) : 0;

      // First compare by version number (higher is better)
      if (aVersion !== bVersion) {
        return bVersion - aVersion;
      }

      // If same version number, check for "v2" or similar in the name
      const aHasV2 = a.modelName.toLowerCase().includes('v2');
      const bHasV2 = b.modelName.toLowerCase().includes('v2');

      if (aHasV2 && !bHasV2) return -1;
      if (!aHasV2 && bHasV2) return 1;

      // If still tied, prefer newer model IDs (they often have dates in them)
      return b.modelId.localeCompare(a.modelId);
    });

    return sortedModels[0];
  }

  /**
   * Implementation of the PackageHandler interface
   */
  public async getLatestVersion(args: any): Promise<{ content: { type: string; text: string }[] }> {
    try {
      let result: BedrockModelSearchResult | BedrockModel | null;

      if (args.action === 'search') {
        result = await this.searchModels(args.query || '', args.provider, args.region);
      } else if (args.action === 'get') {
        const model = await this.getModelById(args.modelId);
        result = {
          models: model ? [model] : [],
          totalCount: model ? 1 : 0
        };
      } else if (args.action === 'get_latest_claude_sonnet') {
        // Get the latest Claude Sonnet model
        const model = await this.getLatestClaudeSonnetModel();

        if (!model) {
          return {
            content: [
              {
                type: 'text',
                text: JSON.stringify({ error: 'No Claude Sonnet model found' }, null, 2)
              }
            ]
          };
        }

        // Return just the model, not wrapped in a search result
        result = model;
      } else {
        // Default to list all models
        result = await this.listModels();
      }

      return {
        content: [
          {
            type: 'text',
            text: JSON.stringify(result, null, 2)
          }
        ]
      };
    } catch (error) {
      console.error('Error in BedrockHandler:', error);
      return {
        content: [
          {
            type: 'text',
            text: `Error fetching Bedrock models: ${error instanceof Error ? error.message : String(error)}`
          }
        ]
      };
    }
  }
}
