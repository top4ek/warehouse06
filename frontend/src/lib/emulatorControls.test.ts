import { describe, expect, it } from "vitest";
import { DEFAULT_CONTROLS, resolveControls } from "./emulatorControls";

describe("emulatorControls", () => {
  it("falls back to the default gamepad without a config", () => {
    expect(resolveControls(null)).toEqual(resolveControls(DEFAULT_CONTROLS));
    expect(resolveControls(undefined)).toEqual(resolveControls(DEFAULT_CONTROLS));
    expect(resolveControls({ rows: [] })).toEqual(resolveControls(DEFAULT_CONTROLS));
  });

  it("resolves named keys to vector06js keycodes", () => {
    const [row] = resolveControls({ rows: [["up", "left", "ss", "space", "f12"]] });
    expect(row.map((b) => b?.keycode)).toEqual([38, 37, 16, 32, 123]);
    expect(row[4]?.edge).toBe(true);
    expect(row[0]?.edge).toBe(false);
  });

  it("resolves single characters to their uppercase char codes", () => {
    const [row] = resolveControls({ rows: [["a", "Z", "5"]] });
    expect(row.map((b) => b?.keycode)).toEqual([65, 90, 53]);
    expect(row.map((b) => b?.label)).toEqual(["A", "Z", "5"]);
  });

  it("keeps empty cells and drops unknown keys", () => {
    const [row] = resolveControls({ rows: [[null, "nope", "up"]] });
    expect(row[0]).toBeNull();
    expect(row[1]).toBeNull();
    expect(row[2]?.keycode).toBe(38);
  });

  it("applies custom labels from object cells", () => {
    const [row] = resolveControls({ rows: [[{ key: "f12", label: "Start" }, { key: "space" }]] });
    expect(row[0]).toEqual({ keycode: 123, label: "Start", edge: true });
    expect(row[1]?.label).toBe("Пробел");
  });
});
