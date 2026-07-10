import DOMPurify from "dompurify";

// Neutralize reverse tabnabbing: any link that keeps target= (e.g. _blank)
// must not expose window.opener to the destination page.
DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "A" && node.hasAttribute("target")) {
    node.setAttribute("rel", "noopener noreferrer");
  }
});

/** Sanitize server-rendered HTML before injecting into the DOM. */
export function sanitizeHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ["target"],
  });
}
