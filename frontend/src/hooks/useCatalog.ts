import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  authorAsEntry,
  listAuthors,
  listEntries,
  searchEntries,
  type Entry,
} from "../api";
import { queryKeys } from "../api/queryKeys";
import { readBrowseFilters, type BrowseFilters } from "../lib/browse";
import type { ListOrigin } from "../lib/listRestore";
import { scrollToListAnchor } from "../lib/listRestore";
import type { SortField, SortOrder } from "../context/PrefsContext";

type CatalogMode = "browse" | "authors";

export const CATALOG_PAGE_SIZE = 20;
export const CATALOG_WINDOW_PAGES = 3;

type Options = {
  mode: CatalogMode;
  fixedAuthor?: string;
  searchKey: string;
  sortField: SortField;
  sortOrder: SortOrder;
  restore?: ListOrigin | null;
};

function pageSpan(start: number, end: number) {
  return end - start + 1;
}

// Captures the viewport position of the entry identified by anchorPath and
// returns a function that restores it after the next render. Anchoring on a
// concrete item keeps the view stable regardless of how the window changed;
// measuring total height deltas breaks down when a page is appended and
// another trimmed in the same update (net delta ~0, view jumps a full page).
function captureAnchor(anchorPath: string): (() => void) | null {
  const find = () =>
    document.querySelector(`[data-entry-path="${CSS.escape(anchorPath)}"]`);
  const anchorEl = find();
  if (!anchorEl) return null;
  const before = anchorEl.getBoundingClientRect().top;
  return () => {
    requestAnimationFrame(() => {
      const after = find()?.getBoundingClientRect().top;
      if (after !== undefined && after !== before) {
        window.scrollBy(0, after - before);
      }
    });
  };
}

export function useCatalog({
  mode,
  fixedAuthor = "",
  searchKey,
  sortField,
  sortOrder,
  restore = null,
}: Options) {
  const queryClient = useQueryClient();
  const pageSize = CATALOG_PAGE_SIZE;
  const [entries, setEntries] = useState<Entry[]>([]);
  const [pageRangeStart, setPageRangeStart] = useState(0);
  const [pageRangeEnd, setPageRangeEnd] = useState(0);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingPrev, setLoadingPrev] = useState(false);
  const [error, setError] = useState("");

  const generationRef = useRef(0);
  const busyRef = useRef(false);
  const pageRangeRef = useRef({ start: 0, end: 0 });
  const totalRef = useRef(0);
  const entriesRef = useRef<Entry[]>([]);
  const restorePendingRef = useRef<ListOrigin | null>(restore ?? null);

  const fetchPage = useCallback(
    async (pageIndex: number, filters: BrowseFilters) => {
      const offset = pageIndex * pageSize;

      if (mode === "authors") {
        const authors = await queryClient.fetchQuery({
          queryKey: queryKeys.authors(),
          queryFn: ({ signal }) => listAuthors(signal),
        });
        const mapped = authors.map(authorAsEntry);
        return { items: mapped.slice(offset, offset + pageSize), total: mapped.length };
      }

      const active = { ...filters, author: fixedAuthor || filters.author };
      if (active.q) {
        return searchEntries(active.q, { limit: pageSize, offset });
      }

      return listEntries({
        limit: pageSize,
        offset,
        sort: sortField,
        order: sortOrder,
        tag: active.tag,
        author: active.author,
        platform: active.platform,
      });
    },
    [mode, pageSize, fixedAuthor, sortField, sortOrder, queryClient],
  );

  const applyWindow = useCallback((start: number, end: number, items: Entry[], itemTotal: number) => {
    pageRangeRef.current = { start, end };
    setPageRangeStart(start);
    setPageRangeEnd(end);
    setEntries(items);
    entriesRef.current = items;
    setTotal(itemTotal);
    totalRef.current = itemTotal;
  }, []);

  const loadRange = useCallback(
    async (rangeStart: number, rangeEnd: number, gen: number) => {
      const filters = readBrowseFilters(searchKey);
      const merged: Entry[] = [];
      const seen = new Set<string>();
      let itemTotal = 0;

      for (let page = rangeStart; page <= rangeEnd; page++) {
        const result = await fetchPage(page, filters);
        if (gen !== generationRef.current) return null;
        itemTotal = result.total;
        for (const item of result.items) {
          if (!seen.has(item.path)) {
            merged.push(item);
            seen.add(item.path);
          }
        }
      }

      return { rangeStart, rangeEnd, merged, itemTotal };
    },
    [searchKey, fetchPage],
  );

  useEffect(() => {
    const gen = ++generationRef.current;
    busyRef.current = false;

    async function run() {
      setLoading(true);
      setLoadingMore(false);
      setLoadingPrev(false);
      setError("");

      try {
        const filters = readBrowseFilters(searchKey);
        const pendingRestore = restorePendingRef.current;
        restorePendingRef.current = null;

        if (pendingRestore) {
          const probe = await fetchPage(0, filters);
          if (gen !== generationRef.current) return;

          const maxPage = Math.max(0, Math.ceil(probe.total / pageSize) - 1);
          const anchor = Math.min(Math.max(0, pendingRestore.anchorPageIndex), maxPage);
          const rangeStart = Math.max(0, anchor - 1);
          const rangeEnd = Math.min(maxPage, anchor + 1);
          const result = await loadRange(rangeStart, rangeEnd, gen);
          if (!result || gen !== generationRef.current) return;

          applyWindow(result.rangeStart, result.rangeEnd, result.merged, result.itemTotal);
          scrollToListAnchor(pendingRestore);
          return;
        }

        const result = await fetchPage(0, filters);
        if (gen !== generationRef.current) return;
        applyWindow(0, 0, result.items, result.total);
      } catch (e: unknown) {
        if (gen === generationRef.current) {
          setError(e instanceof Error ? e.message : "Request failed");
        }
      } finally {
        if (gen === generationRef.current) setLoading(false);
      }
    }

    void run();
  }, [searchKey, sortField, sortOrder, mode, fixedAuthor, fetchPage, loadRange, applyWindow, pageSize]);

  const loadNext = useCallback(async () => {
    if (busyRef.current) return;
    const { start, end } = pageRangeRef.current;
    const itemTotal = totalRef.current;
    const nextPage = end + 1;
    if (nextPage * pageSize >= itemTotal) return;

    busyRef.current = true;
    setLoadingMore(true);
    const gen = generationRef.current;

    try {
      const filters = readBrowseFilters(searchKey);
      const result = await fetchPage(nextPage, filters);
      if (gen !== generationRef.current) return;

      const current = entriesRef.current;
      const seen = new Set(current.map((e) => e.path));
      const appended = result.items.filter((item) => !seen.has(item.path));
      let newStart = start;
      const newEnd = nextPage;
      let newEntries = [...current, ...appended];
      let restoreAnchor: (() => void) | null = null;

      if (pageSpan(newStart, newEnd) > CATALOG_WINDOW_PAGES) {
        newEntries = newEntries.slice(pageSize);
        newStart += 1;
        // Trimming the top page shifts everything up; keep the first
        // remaining entry where the user sees it now.
        restoreAnchor = newEntries[0] ? captureAnchor(newEntries[0].path) : null;
      }

      applyWindow(newStart, newEnd, newEntries, result.total);
      restoreAnchor?.();
    } catch (e: unknown) {
      if (gen === generationRef.current) {
        setError(e instanceof Error ? e.message : "Request failed");
      }
    } finally {
      busyRef.current = false;
      setLoadingMore(false);
    }
  }, [searchKey, fetchPage, pageSize, applyWindow]);

  const loadPrev = useCallback(async () => {
    if (busyRef.current) return;
    const { start, end } = pageRangeRef.current;
    if (start === 0) return;

    busyRef.current = true;
    setLoadingPrev(true);
    const gen = generationRef.current;

    try {
      const filters = readBrowseFilters(searchKey);
      const prevPage = start - 1;
      const result = await fetchPage(prevPage, filters);
      if (gen !== generationRef.current) return;

      const current = entriesRef.current;
      const seen = new Set(current.map((e) => e.path));
      const prepended = result.items.filter((item) => !seen.has(item.path));
      const newStart = prevPage;
      let newEnd = end;
      let newEntries = [...prepended, ...current];
      // Content is inserted above the viewport; keep the current first entry
      // where the user sees it now (it survives an end-trim).
      const restoreAnchor = current[0] ? captureAnchor(current[0].path) : null;

      if (pageSpan(newStart, newEnd) > CATALOG_WINDOW_PAGES) {
        newEntries = newEntries.slice(0, newEntries.length - pageSize);
        newEnd -= 1;
      }

      applyWindow(newStart, newEnd, newEntries, result.total);
      restoreAnchor?.();
    } catch (e: unknown) {
      if (gen === generationRef.current) {
        setError(e instanceof Error ? e.message : "Request failed");
      }
    } finally {
      busyRef.current = false;
      setLoadingPrev(false);
    }
  }, [searchKey, fetchPage, pageSize, applyWindow]);

  const hasMore = (pageRangeEnd + 1) * pageSize < total;
  const hasPrev = pageRangeStart > 0;

  return {
    entries,
    total,
    pageRangeStart,
    pageRangeEnd,
    loading,
    loadingMore,
    loadingPrev,
    error,
    hasMore,
    hasPrev,
    loadNext,
    loadPrev,
  };
}
