import { theme } from "antd";
import type { ThemeConfig } from "antd";

export const THEME_STORAGE_KEY = "warehouse06-theme";

export type ThemeMode = "light" | "dark";

const baseToken = {
  borderRadius: 12,
  fontFamily:
    '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
};

export function getThemeConfig(mode: ThemeMode): ThemeConfig {
  return {
    algorithm: mode === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm,
    cssVar: { key: "warehouse06" },
    token: baseToken,
    components: {
      Card: {
        paddingLG: 24,
      },
    },
  };
}

export function loadInitialThemeMode(): ThemeMode {
  const saved = localStorage.getItem(THEME_STORAGE_KEY);
  if (saved === "light" || saved === "dark") return saved;
  if (window.matchMedia("(prefers-color-scheme: dark)").matches) return "dark";
  return "light";
}
