import { test, expect, type Page } from "@playwright/test";

// Regression test for the bidirectional infinite scroll on /browse.
// The catalog keeps a 3-page window (CATALOG_WINDOW_PAGES) of 20-item pages.
// Historically, once the view was pinned against the top or bottom edge the
// next page would not load until the user jiggled the scroll position:
// the IntersectionObserver sentinel stayed inside its rootMargin (no
// transition => no callback) and the height-delta scroll compensation
// cancelled out to ~0 when a page was appended and another trimmed.
//
// The API is mocked, so the test is hermetic: 100 synthetic entries,
// 5 pages of 20.

const PAGE_SIZE = 20;
const TOTAL = 100;

function makeEntry(index: number) {
  const n = index + 1;
  return {
    id: n,
    path: `authors/author${Math.floor(index / 10) + 1}/entry-${n}`,
    name: `Entry ${n}`,
    description: `Synthetic description ${n}`,
  };
}

async function mockApi(page: Page) {
  await page.route("**/api/status", (route) => route.fulfill({ json: { syncing: false } }));
  await page.route("**/api/tags", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/authors", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/platforms", (route) => route.fulfill({ json: [] }));
  await page.route(/\/api\/entries\?/, (route) => {
    const url = new URL(route.request().url());
    const offset = Number(url.searchParams.get("offset") ?? "0");
    const limit = Number(url.searchParams.get("limit") ?? String(PAGE_SIZE));
    const count = Math.max(0, Math.min(limit, TOTAL - offset));
    const items = Array.from({ length: count }, (_, k) => makeEntry(offset + k));
    return route.fulfill({ json: { items, total: TOTAL } });
  });
}

function entryText(page: Page, n: number) {
  return page.getByText(`Entry ${n}`, { exact: true });
}

async function pinToBottom(page: Page) {
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
}

async function pinToTop(page: Page) {
  await page.evaluate(() => window.scrollTo(0, 0));
}

async function scrollToLastPage(page: Page) {
  // Walk down page by page; each step pins the view to the absolute bottom
  // and expects the next page to arrive without any jiggling.
  // Markers 21/41 fill the window; 61/81 also force a trim of the top page.
  for (const marker of [21, 41, 61, 81]) {
    await pinToBottom(page);
    await expect(entryText(page, marker)).toBeAttached({ timeout: 10_000 });
  }
}

test.beforeEach(async ({ page }) => {
  await mockApi(page);
  await page.goto("/browse");
  await expect(entryText(page, 1)).toBeVisible();
});

test("loads next pages while pinned to the bottom edge", async ({ page }) => {
  await scrollToLastPage(page);
  await expect(entryText(page, TOTAL)).toBeAttached();
});

test("loads previous pages while pinned to the top edge", async ({ page }) => {
  await scrollToLastPage(page);

  // Window now holds pages 2..4 (entries 41-100). Walk back up.
  await pinToTop(page);
  await expect(entryText(page, 21)).toBeAttached({ timeout: 10_000 });

  await pinToTop(page);
  await expect(entryText(page, 1)).toBeAttached({ timeout: 10_000 });
});

test("keeps the viewport anchored when the window is trimmed", async ({ page }) => {
  // Fill the window: pages 0..2 (entries 1-60), no trim yet.
  for (const marker of [21, 41]) {
    await pinToBottom(page);
    await expect(entryText(page, marker)).toBeAttached({ timeout: 10_000 });
  }

  // Remember which entry sits at the top of the viewport, then trigger a
  // load that trims the first page. The same entry must stay in view
  // (no page-sized jump).
  await pinToBottom(page);
  await expect(entryText(page, 61)).toBeAttached({ timeout: 10_000 });

  const anchored = await page.evaluate(() => {
    const items = Array.from(document.querySelectorAll<HTMLElement>("[data-entry-path]"));
    const topItem = items.find((el) => el.getBoundingClientRect().bottom > 0);
    return topItem?.dataset.entryPath ?? null;
  });
  expect(anchored).not.toBeNull();

  // After the trim settles, the anchored entry should still be near the
  // viewport (within one viewport height), not a full page away.
  await page.waitForTimeout(300);
  const offset = await page.evaluate((path) => {
    const el = document.querySelector(`[data-entry-path="${path}"]`);
    if (!el) return Number.POSITIVE_INFINITY;
    return Math.abs(el.getBoundingClientRect().top);
  }, anchored);
  expect(offset).toBeLessThan(page.viewportSize()!.height * 1.5);
});
