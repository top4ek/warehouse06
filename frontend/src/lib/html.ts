/** First <img src="..."> from HTML content (platform README, etc.). */
export function firstImageSrc(html?: string): string | null {
  if (!html) return null;
  const doc = new DOMParser().parseFromString(html, "text/html");
  const src = doc.querySelector("img")?.getAttribute("src");
  if (!src) return null;
  // Allow http(s) and relative paths; reject javascript:, data:, etc.
  if (/^[a-z][a-z0-9+.-]*:/i.test(src) && !/^https?:/i.test(src)) return null;
  return src;
}

/** Resolve image path from entry HTML relative to storage base path. */
export function resolveStorageImage(src: string, basePath: string): string {
  if (src.startsWith("http://") || src.startsWith("https://")) return src;
  const clean = src.replace(/^\.\//, "").replace(/^\/+/, "");
  if (clean.includes("/")) return clean;
  return `${basePath.replace(/\/+$/, "")}/${clean}`;
}
