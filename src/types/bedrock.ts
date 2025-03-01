export interface BedrockModel {
  provider: string;
  modelName: string;
  modelId: string;
  regionsSupported: string[];
  inputModalities: string[];
  outputModalities: string[];
  streamingSupported: boolean;
}

export interface BedrockModelSearchResult {
  models: BedrockModel[];
  totalCount: number;
}
