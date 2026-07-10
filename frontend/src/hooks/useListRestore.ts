import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import {
  clearListOrigin,
  readListOrigin,
  scrollToListAnchor,
  type ListOrigin,
} from "../lib/listRestore";

export function useListRestore() {
  const location = useLocation();
  const [initialRestore] = useState<ListOrigin | null>(() => {
    const fromState = (location.state as { restore?: ListOrigin } | null)?.restore;
    if (fromState) return fromState;
    const stored = readListOrigin();
    if (stored && stored.pathname === location.pathname && stored.search === location.search) {
      return stored;
    }
    return null;
  });

  useEffect(() => {
    if (!initialRestore) return;
    clearListOrigin();
    scrollToListAnchor(initialRestore);
  }, [initialRestore]);

  return initialRestore;
}
