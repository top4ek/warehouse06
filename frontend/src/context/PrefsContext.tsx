import { createContext, useContext, useMemo, type ReactNode } from "react";
import { usePersistentState } from "../hooks/usePersistentState";

export type SortField = "date" | "created_at" | "name";
export type SortOrder = "asc" | "desc";
export type ViewMode = "compact" | "comfortable" | "grid";

const SORT_FIELD_KEY = "warehouse06:sortField";
const SORT_ORDER_KEY = "warehouse06:sortOrder";
const VIEW_KEY = "warehouse06:viewMode";

const SORT_FIELDS: SortField[] = ["date", "created_at", "name"];
const SORT_ORDERS: SortOrder[] = ["asc", "desc"];
const VIEW_MODES: ViewMode[] = ["compact", "comfortable", "grid"];

export const SORT_FIELD_LABELS: Record<SortField, string> = {
  date: "Creation date",
  created_at: "Date added",
  name: "Name",
};

export const SORT_ORDER_LABELS: Record<SortOrder, string> = {
  asc: "Ascending",
  desc: "Descending",
};

function oneOf<T extends string>(allowed: T[]) {
  return (raw: string): T | null => (allowed.includes(raw as T) ? (raw as T) : null);
}

type PrefsContextValue = {
  sortField: SortField;
  sortOrder: SortOrder;
  viewMode: ViewMode;
  setSortField: (field: SortField) => void;
  setSortOrder: (order: SortOrder) => void;
  setViewMode: (mode: ViewMode) => void;
};

const PrefsContext = createContext<PrefsContextValue | null>(null);

export function PrefsProvider({ children }: { children: ReactNode }) {
  const [sortField, setSortField] = usePersistentState<SortField>(
    SORT_FIELD_KEY,
    () => "created_at",
    oneOf(SORT_FIELDS),
  );
  const [sortOrder, setSortOrder] = usePersistentState<SortOrder>(
    SORT_ORDER_KEY,
    () => "desc",
    oneOf(SORT_ORDERS),
  );
  const [viewMode, setViewMode] = usePersistentState<ViewMode>(
    VIEW_KEY,
    () => "comfortable",
    oneOf(VIEW_MODES),
  );

  const value = useMemo(
    () => ({
      sortField,
      sortOrder,
      viewMode,
      setSortField,
      setSortOrder,
      setViewMode,
    }),
    [sortField, sortOrder, viewMode, setSortField, setSortOrder, setViewMode],
  );

  return <PrefsContext.Provider value={value}>{children}</PrefsContext.Provider>;
}

export function usePrefs() {
  const ctx = useContext(PrefsContext);
  if (!ctx) throw new Error("usePrefs must be used within PrefsProvider");
  return ctx;
}
