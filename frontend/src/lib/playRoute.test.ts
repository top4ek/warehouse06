import { describe, expect, it } from "vitest";
import {
  entryPlayHref,
  entryPlayLocation,
  isReservedEntryPath,
  parsePlayHash,
  parsePlaySearch,
  parsePlaySplat,
} from "./playRoute";

describe("playRoute", () => {
  it("builds play href and location", () => {
    expect(entryPlayHref("vector06c/airforce", "airforce.zip")).toBe(
      "/vector06c/airforce?play=airforce.zip",
    );
    expect(entryPlayLocation("vector06c/airforce", "airforce.zip")).toEqual({
      pathname: "/vector06c/airforce",
      search: "?play=airforce.zip",
    });
  });

  it("parses play query fragments", () => {
    expect(parsePlaySearch("?play=airforce.zip")).toBe("airforce.zip");
    expect(parsePlayHash("#play=exolon.rom")).toBe("exolon.rom");
    expect(parsePlaySplat("vector06c/exolon/play/exolon.rom")).toEqual({
      entryPath: "vector06c/exolon",
      filename: "exolon.rom",
    });
  });

  it("detects reserved entry paths", () => {
    expect(isReservedEntryPath("emulator/")).toBe(true);
    expect(isReservedEntryPath("vector06c/airforce")).toBe(false);
  });
});
