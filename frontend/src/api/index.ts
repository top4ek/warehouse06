import { getJSON } from "./client";
import { isReservedEntryPath } from "../lib/playRoute";
import {
  authorListSchema,
  authorSchema,
  entryListResultSchema,
  entrySchema,
  platformListSchema,
  syncStatusSchema,
  tagListSchema,
} from "./schemas";
import type { Author, Entry } from "./types";

export type {
  Author,
  DirectoryItem,
  Entry,
  EntryListResult,
  FileItem,
  Platform,
  StorageCommit,
  SyncStatus,
  Tag,
} from "./types";
export { absoluteStorageUrl, storageUrl } from "./client";

type QueryValue = string | number | undefined | null;

export function listEntries(params?: Record<string, QueryValue>, signal?: AbortSignal) {
  return getJSON("/api/entries", params, entryListResultSchema, signal);
}

export function searchEntries(
  query: string,
  params?: Record<string, QueryValue>,
  signal?: AbortSignal,
) {
  return getJSON("/api/entries/search", { q: query, ...params }, entryListResultSchema, signal);
}

export function getEntry(path: string, signal?: AbortSignal) {
  if (isReservedEntryPath(path)) {
    return Promise.reject(new Error("Not found"));
  }
  return getJSON(`/api/entries/${encodeURIComponent(path)}`, undefined, entrySchema, signal);
}

export function listAuthors(signal?: AbortSignal) {
  return getJSON("/api/authors", undefined, authorListSchema, signal);
}

export function getAuthor(dir: string, signal?: AbortSignal): Promise<Author> {
  return getJSON(`/api/authors/${encodeURIComponent(dir)}`, undefined, authorSchema, signal);
}

export function listTags(signal?: AbortSignal) {
  return getJSON("/api/tags", undefined, tagListSchema, signal);
}

export function listPlatforms(signal?: AbortSignal) {
  return getJSON("/api/platforms", undefined, platformListSchema, signal);
}

export function getSyncStatus(signal?: AbortSignal) {
  return getJSON("/api/status", undefined, syncStatusSchema, signal);
}

export function authorAsEntry(author: Author): Entry {
  return {
    id: author.id,
    path: `authors/${author.directory_name}`,
    name: author.name || author.directory_name,
    description: author.address || "",
    preview_image: author.preview_image,
    platform: "Authors",
    entry_count: author.entry_count,
  };
}
