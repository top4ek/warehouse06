export const queryKeys = {
  entries: (params: Record<string, unknown>) => ["entries", params] as const,
  entry: (path: string) => ["entry", path] as const,
  authors: () => ["authors"] as const,
  author: (dir: string) => ["author", dir] as const,
  tags: () => ["tags"] as const,
  platforms: () => ["platforms"] as const,
  syncStatus: () => ["syncStatus"] as const,
};
