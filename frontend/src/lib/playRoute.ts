const PLAY_HASH_PREFIX = "play=";
const PLAY_SEGMENT = "/play/";
const PLAY_SEARCH_PARAM = "play";

function normalizeEntryPath(entryPath: string): string {
  return entryPath.replace(/^\/+|\/+$/g, "");
}

/** Legacy hash URLs: #play=filename */
export function entryPlayHash(filename: string): string {
  return `${PLAY_HASH_PREFIX}${encodeURIComponent(filename)}`;
}

/** Entry page URL that opens the emulator, e.g. /vector06c/764?play=512demo.r0m */
export function entryPlayHref(entryPath: string, filename: string): string {
  const base = normalizeEntryPath(entryPath);
  if (!base) {
    return `/?${PLAY_SEARCH_PARAM}=${encodeURIComponent(filename)}`;
  }
  return `/${base}?${PLAY_SEARCH_PARAM}=${encodeURIComponent(filename)}`;
}

/** Location object for react-router navigate(). */
export function entryPlayLocation(entryPath: string, filename: string) {
  const base = normalizeEntryPath(entryPath);
  const pathname = base ? `/${base}` : "/";
  return {
    pathname,
    search: `?${PLAY_SEARCH_PARAM}=${encodeURIComponent(filename)}`,
  };
}

export function parsePlaySearch(search: string): string | null {
  const raw = search.startsWith("?") ? search.slice(1) : search;
  const value = new URLSearchParams(raw).get(PLAY_SEARCH_PARAM);
  if (!value) return null;
  try {
    return decodeURIComponent(value);
  } catch {
    return null;
  }
}

/** Legacy hash URLs: #play=filename */
export function parsePlayHash(hash: string): string | null {
  const h = hash.replace(/^#/, "");
  if (!h.startsWith(PLAY_HASH_PREFIX)) return null;
  const value = h.slice(PLAY_HASH_PREFIX.length);
  if (!value) return null;
  try {
    return decodeURIComponent(value);
  } catch {
    return null;
  }
}

/** Legacy pathname URLs: /entry/path/play/file.rom */
export function parsePlaySplat(splat: string): { entryPath: string; filename: string } | null {
  const normalized = splat.replace(/^\/+|\/+$/g, "");
  const idx = normalized.lastIndexOf(PLAY_SEGMENT);
  if (idx === -1) return null;
  const entryPath = normalized.slice(0, idx);
  const encodedFile = normalized.slice(idx + PLAY_SEGMENT.length);
  if (!entryPath || !encodedFile) return null;
  try {
    return { entryPath, filename: decodeURIComponent(encodedFile) };
  } catch {
    return null;
  }
}

/** Paths that must not be loaded as catalog entries (static app / API). */
export function isReservedEntryPath(path: string): boolean {
  const normalized = normalizeEntryPath(path);
  if (!normalized) return true;
  if (normalized === "emulator" || normalized.startsWith("emulator/")) return true;
  if (normalized === "api" || normalized.startsWith("api/")) return true;
  return false;
}
