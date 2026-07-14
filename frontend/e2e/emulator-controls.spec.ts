import { test, expect, type Frame, type Page } from "@playwright/test";

// On-screen gamepad for the emulator dialog: on a mobile viewport the panel
// is visible by default, its buttons post key events into the emulator
// iframe (the real vector06js embedded-input bridge), the layout comes from
// the entry's `controls` frontmatter (with a built-in default), and the
// titlebar toggler shows/hides the panel.

test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });

const baseEntry = {
  id: 1,
  path: "vector06c/game",
  name: "Game",
  description: "",
  content_html: "<p>game</p>",
  files: [{ id: 1, filename: "game.rom", filepath: "vector06c/game/game.rom", is_image: false }],
};

async function mockApi(page: Page, entry: Record<string, unknown>) {
  await page.route("**/api/status", (route) => route.fulfill({ json: { syncing: false } }));
  await page.route("**/api/tags", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/authors", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/platforms", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/entries/**", (route) => route.fulfill({ json: entry }));
  // Anchored to the path: the emulator iframe URL also *ends* with the ROM
  // name (/emulator/?i:<rom url>) and must not be swallowed by this mock.
  await page.route(/^https?:\/\/[^/]+\/vector06c\/game\/game\.rom$/, (route) =>
    route.fulfill({ contentType: "application/octet-stream", body: Buffer.alloc(256) }),
  );
}

async function openEmulator(page: Page): Promise<Frame> {
  await page.goto("/vector06c/game?play=game.rom");
  const iframeEl = await page.waitForSelector("iframe.ui-window__emulator-frame");
  const frame = await iframeEl.contentFrame();
  expect(frame).not.toBeNull();
  // The embedded-input bridge (emulator-src/src/main.js) sets this flag right
  // as it registers its message listener; the frame's load event is unreliable
  // here because emulator subresources can keep loading indefinitely.
  await frame!.waitForFunction(
    () => (window as unknown as { __v06EmbeddedInputBridge?: boolean }).__v06EmbeddedInputBridge,
  );
  return frame!;
}

function collectInputMessages(frame: Frame) {
  return frame.evaluate(() => {
    const win = window as unknown as { __received?: unknown[]; __collecting?: boolean };
    win.__received = [];
    if (win.__collecting) return;
    win.__collecting = true;
    window.addEventListener("message", (e: MessageEvent) => {
      if (e.data?.cmd === "input") win.__received!.push(e.data);
    });
  });
}

function receivedMessages(frame: Frame) {
  return frame.evaluate(() => (window as unknown as { __received?: unknown[] }).__received ?? []);
}

test("custom controls layout sends configured keys into the emulator", async ({ page }) => {
  await mockApi(page, {
    ...baseEntry,
    controls: {
      rows: [
        [null, "up", null, null, { key: "f12", label: "Start" }],
        ["left", "down", "right", null, "space"],
      ],
    },
  });

  const frame = await openEmulator(page);
  const panel = page.locator(".emulator-controls");
  await expect(panel).toBeVisible();
  await expect(panel.getByRole("button", { name: "Start" })).toBeVisible();

  await collectInputMessages(frame);
  await panel.getByRole("button", { name: "▶" }).tap();

  await expect
    .poll(() => receivedMessages(frame))
    .toEqual([
      { cmd: "input", subcmd: "keydown", keycode: 39 },
      { cmd: "input", subcmd: "keyup", keycode: 39 },
    ]);

  // F12 is edge-triggered: keydown only.
  await collectInputMessages(frame);
  await panel.getByRole("button", { name: "Start" }).tap();
  await expect
    .poll(() => receivedMessages(frame))
    .toEqual([{ cmd: "input", subcmd: "keydown", keycode: 123 }]);
});

test("default gamepad shows without controls frontmatter and toggler hides it", async ({
  page,
}) => {
  await mockApi(page, baseEntry);

  await openEmulator(page);
  const panel = page.locator(".emulator-controls");
  await expect(panel).toBeVisible();
  await expect(panel.getByRole("button", { name: "БЛК+СБР" })).toBeVisible();
  await expect(panel.getByRole("button", { name: "Пробел" })).toBeVisible();

  const toggler = page.getByRole("button", { name: "On-screen controls" });
  await toggler.tap();
  await expect(panel).toHaveCount(0);
  await toggler.tap();
  await expect(panel).toBeVisible();
});
