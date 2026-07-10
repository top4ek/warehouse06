import { describe, expect, it } from "vitest";
import { sanitizeHtml } from "./sanitizeHtml";

describe("sanitizeHtml", () => {
  it("keeps safe markup", () => {
    expect(sanitizeHtml("<p>Hello <strong>world</strong></p>")).toBe(
      "<p>Hello <strong>world</strong></p>",
    );
  });

  it("removes script tags", () => {
    expect(sanitizeHtml('<p>ok</p><script>alert(1)</script>')).toBe("<p>ok</p>");
  });

  it("allows target attribute on links", () => {
    expect(sanitizeHtml('<a href="https://example.com" target="_blank">link</a>')).toContain(
      'target="_blank"',
    );
  });

  it("forces rel=noopener on links with target", () => {
    const out = sanitizeHtml('<a href="https://example.com" target="_blank">link</a>');
    expect(out).toContain('rel="noopener noreferrer"');
  });

  it("overrides an attacker-supplied rel when target is present", () => {
    const out = sanitizeHtml('<a href="https://evil.example" target="_blank" rel="opener">x</a>');
    expect(out).toContain('rel="noopener noreferrer"');
    expect(out).not.toContain('rel="opener"');
  });
});
