import { beforeEach, describe, expect, it } from "vitest";
import {
  clearListOrigin,
  listOriginLabel,
  readListOrigin,
  saveListOrigin,
  type ListOrigin,
} from "./listRestore";

const origin: ListOrigin = {
  pathname: "/browse",
  search: "?tag=demo",
  anchorPageIndex: 2,
  anchorEntryPath: "authors/ivan/entry-42",
  scrollY: 1234,
};

beforeEach(() => {
  sessionStorage.clear();
});

describe("saveListOrigin / readListOrigin", () => {
  it("round-trips an origin through sessionStorage", () => {
    saveListOrigin(origin);
    expect(readListOrigin()).toEqual(origin);
  });

  it("returns null when nothing is stored", () => {
    expect(readListOrigin()).toBeNull();
  });

  it("returns null for corrupt JSON", () => {
    sessionStorage.setItem("warehouse06:listOrigin", "{not json");
    expect(readListOrigin()).toBeNull();
  });

  it("returns null for a stale schema", () => {
    sessionStorage.setItem(
      "warehouse06:listOrigin",
      JSON.stringify({ pathname: "/browse", page: 2 }),
    );
    expect(readListOrigin()).toBeNull();
  });

  it("clears the stored origin", () => {
    saveListOrigin(origin);
    clearListOrigin();
    expect(readListOrigin()).toBeNull();
  });
});

describe("listOriginLabel", () => {
  it("labels home, author, and list origins", () => {
    expect(listOriginLabel({ ...origin, pathname: "/" })).toBe("Back to home");
    expect(listOriginLabel({ ...origin, pathname: "/authors/ivan" })).toBe("Back to author works");
    expect(listOriginLabel(origin)).toBe("Back to list");
  });
});
