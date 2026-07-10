import { describe, expect, it } from "vitest";
import { browsePath, readBrowseFilters, type BrowseFilters } from "./browse";

describe("readBrowseFilters", () => {
  it("reads all supported params", () => {
    expect(readBrowseFilters("?q=cat&tag=game&author=ivan&platform=vector06")).toEqual({
      q: "cat",
      tag: "game",
      author: "ivan",
      platform: "vector06",
    });
  });

  it("returns undefined for missing params", () => {
    expect(readBrowseFilters("")).toEqual({
      q: undefined,
      tag: undefined,
      author: undefined,
      platform: undefined,
    });
  });

  it("ignores unknown params", () => {
    expect(readBrowseFilters("?foo=bar&tag=demo").tag).toBe("demo");
  });
});

describe("browsePath", () => {
  it("returns plain /browse for empty filters", () => {
    expect(browsePath({})).toBe("/browse");
  });

  it("omits empty-string filters", () => {
    expect(browsePath({ q: "", tag: "demo" })).toBe("/browse?tag=demo");
  });

  it("round-trips filters through the query string", () => {
    const filters: BrowseFilters = {
      q: "hello world",
      tag: "с русским",
      author: "ivan",
      platform: "vector06",
    };
    const path = browsePath(filters);
    const search = path.slice(path.indexOf("?"));
    expect(readBrowseFilters(search)).toEqual(filters);
  });
});
