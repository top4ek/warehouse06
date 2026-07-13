import type { ControlsConfig } from "../api/types";

export type ControlButton = {
  keycode: number;
  label: string;
  /** F11/F12 are edge-triggered in the emulator bridge: keydown only. */
  edge: boolean;
};

/** Named keys usable in `controls` frontmatter, mapped to vector06js host keycodes. */
const NAMED_KEYS: Record<string, { code: number; label: string; edge?: boolean }> = {
  up: { code: 38, label: "▲" },
  down: { code: 40, label: "▼" },
  left: { code: 37, label: "◀" },
  right: { code: 39, label: "▶" },
  space: { code: 32, label: "Пробел" },
  enter: { code: 13, label: "ВК" },
  tab: { code: 9, label: "ТАБ" },
  esc: { code: 27, label: "АР2" },
  backspace: { code: 8, label: "ЗБ" },
  ss: { code: 16, label: "СС" },
  us: { code: 17, label: "УС" },
  ps: { code: 18, label: "ПС" },
  rus: { code: 117, label: "РУС/ЛАТ" },
  f11: { code: 122, label: "БЛК+ВВОД", edge: true },
  f12: { code: 123, label: "БЛК+СБР", edge: true },
};

export const DEFAULT_CONTROLS: ControlsConfig = {
  rows: [
    [null, "up", null, "f11", "f12"],
    ["left", "down", "right", "ss", "space"],
  ],
};

function resolveKey(name: string): ControlButton | null {
  const norm = name.trim().toLowerCase();
  const named = NAMED_KEYS[norm];
  if (named) return { keycode: named.code, label: named.label, edge: named.edge ?? false };
  if (/^[a-z0-9]$/.test(norm)) {
    const upper = norm.toUpperCase();
    return { keycode: upper.charCodeAt(0), label: upper, edge: false };
  }
  return null;
}

/** Expands a `controls` config (or the default gamepad) into renderable rows. */
export function resolveControls(
  config: ControlsConfig | null | undefined,
): (ControlButton | null)[][] {
  const rows = config?.rows?.length ? config.rows : DEFAULT_CONTROLS.rows;
  return rows.map((row) =>
    row.map((cell) => {
      if (cell == null) return null;
      const name = typeof cell === "string" ? cell : cell.key;
      const button = resolveKey(name);
      if (!button) return null;
      const label = typeof cell === "string" ? undefined : cell.label;
      return label ? { ...button, label } : button;
    }),
  );
}
