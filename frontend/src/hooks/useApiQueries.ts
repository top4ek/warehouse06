import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getAuthor,
  getEntry,
  getSyncStatus,
  listAuthors,
  listEntries,
  listPlatforms,
  listTags,
} from "../api";
import { queryKeys } from "../api/queryKeys";

const LATEST_SORT = { sort: "created_at" as const, order: "desc" as const, limit: 12 };

// Tags/authors/platforms only change when the archive re-syncs, so keep them
// fresh for long; useSyncStatus invalidates the cache on sync completion.
const REFERENCE_STALE_TIME = 10 * 60_000;

export function useLatestEntries() {
  return useQuery({
    queryKey: queryKeys.entries(LATEST_SORT),
    queryFn: ({ signal }) => listEntries(LATEST_SORT, signal),
  });
}

export function useTags() {
  return useQuery({
    queryKey: queryKeys.tags(),
    queryFn: ({ signal }) => listTags(signal),
    staleTime: REFERENCE_STALE_TIME,
  });
}

export function useAuthors() {
  return useQuery({
    queryKey: queryKeys.authors(),
    queryFn: ({ signal }) => listAuthors(signal),
    staleTime: REFERENCE_STALE_TIME,
  });
}

export function usePlatforms() {
  return useQuery({
    queryKey: queryKeys.platforms(),
    queryFn: ({ signal }) => listPlatforms(signal),
    staleTime: REFERENCE_STALE_TIME,
  });
}

export function useSyncStatus() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: queryKeys.syncStatus(),
    queryFn: ({ signal }) => getSyncStatus(signal),
    refetchInterval: (q) => (q.state.data?.syncing ? 3000 : false),
  });

  // When a sync finishes, every dataset may have changed; drop the cache.
  const syncing = query.data?.syncing;
  const prevSyncingRef = useRef(syncing);
  useEffect(() => {
    if (prevSyncingRef.current === true && syncing === false) {
      void queryClient.invalidateQueries();
    }
    prevSyncingRef.current = syncing;
  }, [syncing, queryClient]);

  return query;
}

export function useEntry(path: string, enabled = true) {
  return useQuery({
    queryKey: queryKeys.entry(path),
    queryFn: ({ signal }) => getEntry(path, signal),
    enabled: enabled && Boolean(path),
  });
}

export function useAuthor(dir: string, enabled = true) {
  return useQuery({
    queryKey: queryKeys.author(dir),
    queryFn: ({ signal }) => getAuthor(dir, signal),
    enabled: enabled && Boolean(dir),
  });
}
