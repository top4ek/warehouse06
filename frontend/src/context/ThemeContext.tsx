import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  type ReactNode,
} from "react";
import { loadInitialThemeMode, THEME_STORAGE_KEY, type ThemeMode } from "../theme";
import { applyFavicon } from "../theme/favicon";
import { usePersistentState } from "../hooks/usePersistentState";

type ThemeContextValue = {
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  toggleMode: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = usePersistentState<ThemeMode>(
    THEME_STORAGE_KEY,
    loadInitialThemeMode,
    (raw) => (raw === "dark" || raw === "light" ? raw : null),
  );

  const toggleMode = useCallback(() => {
    setMode(mode === "dark" ? "light" : "dark");
  }, [mode, setMode]);

  useEffect(() => {
    document.documentElement.setAttribute("data-warehouse06", mode);
    applyFavicon(mode);
  }, [mode]);

  const value = useMemo(
    () => ({ mode, setMode, toggleMode }),
    [mode, setMode, toggleMode],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useThemeMode() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useThemeMode must be used within ThemeProvider");
  return ctx;
}
