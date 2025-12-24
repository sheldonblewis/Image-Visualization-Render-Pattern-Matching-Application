export type QueryMode = 'percent' | 'regex';

export interface QueryRequest {
  pattern: string;
  mode: QueryMode;
  pageSize: number;
  cursor?: string | null;
}

export interface QueryItem {
  object: string;
  url: string;
  captures: Record<string, string>;
}

export interface QueryResponse {
  captureNames: string[];
  items: QueryItem[];
  nextCursor?: string | null;
  stats?: {
    scannedPrefixes: number;
    scannedObjects: number;
    matched: number;
  };
}
