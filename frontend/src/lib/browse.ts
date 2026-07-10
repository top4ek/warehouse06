export type BrowseFilters = {
  q?: string;
  tag?: string;
  author?: string;
  platform?: string;
};

export function readBrowseFilters(search: string): BrowseFilters {
  const params = new URLSearchParams(search);
  return {
    q: params.get("q") ?? undefined,
    tag: params.get("tag") ?? undefined,
    author: params.get("author") ?? undefined,
    platform: params.get("platform") ?? undefined,
  };
}

export function browsePath(filters: BrowseFilters): string {
  const params = new URLSearchParams();
  if (filters.q) params.set("q", filters.q);
  if (filters.tag) params.set("tag", filters.tag);
  if (filters.author) params.set("author", filters.author);
  if (filters.platform) params.set("platform", filters.platform);
  const query = params.toString();
  return query ? `/browse?${query}` : "/browse";
}
