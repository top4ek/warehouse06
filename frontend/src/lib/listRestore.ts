import { z } from "zod";

const STORAGE_KEY = "warehouse06:listOrigin";

// Stored state survives deploys; validate instead of trusting the cast so a
// schema change cannot feed NaN/undefined into scroll restoration.
const listOriginSchema = z.object({
  pathname: z.string(),
  search: z.string(),
  anchorPageIndex: z.number(),
  anchorEntryPath: z.string(),
  scrollY: z.number(),
});

export type ListOrigin = z.infer<typeof listOriginSchema>;

export function saveListOrigin(origin: ListOrigin): void {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(origin));
}

export function readListOrigin(): ListOrigin | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = listOriginSchema.safeParse(JSON.parse(raw));
    return parsed.success ? parsed.data : null;
  } catch {
    return null;
  }
}

export function clearListOrigin(): void {
  sessionStorage.removeItem(STORAGE_KEY);
}

export function listOriginLabel(origin: ListOrigin): string {
  if (origin.pathname === "/") return "Back to home";
  if (origin.pathname.startsWith("/authors/")) return "Back to author works";
  return "Back to list";
}

export function scrollToListAnchor(restore: Pick<ListOrigin, "anchorEntryPath" | "scrollY">): void {
  requestAnimationFrame(() => {
    const el = document.querySelector(`[data-entry-path="${CSS.escape(restore.anchorEntryPath)}"]`);
    if (el) {
      el.scrollIntoView({ block: "center" });
      return;
    }
    window.scrollTo(0, restore.scrollY);
  });
}
