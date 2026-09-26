import { expect, test } from "@playwright/test";
import { crc32, deflateRawSync } from "node:zlib";

// A deflated archive exercises zip.js workers and decompression, not just its directory parser.
function archive(name: string, contents: Buffer): Buffer {
  const filename = Buffer.from(name);
  const compressed = deflateRawSync(contents);
  const checksum = crc32(contents);
  const local = Buffer.alloc(30);
  local.writeUInt32LE(0x04034b50);
  local.writeUInt16LE(20, 4);
  local.writeUInt16LE(8, 8);
  local.writeUInt32LE(checksum, 14);
  local.writeUInt32LE(compressed.length, 18);
  local.writeUInt32LE(contents.length, 22);
  local.writeUInt16LE(filename.length, 26);
  const central = Buffer.alloc(46);
  central.writeUInt32LE(0x02014b50);
  central.writeUInt16LE(20, 4);
  central.writeUInt16LE(20, 6);
  central.writeUInt16LE(8, 10);
  central.writeUInt32LE(checksum, 16);
  central.writeUInt32LE(compressed.length, 20);
  central.writeUInt32LE(contents.length, 24);
  central.writeUInt16LE(filename.length, 28);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50);
  end.writeUInt16LE(1, 8);
  end.writeUInt16LE(1, 10);
  end.writeUInt32LE(central.length + filename.length, 12);
  end.writeUInt32LE(local.length + filename.length + compressed.length, 16);
  return Buffer.concat([local, filename, compressed, central, filename, end]);
}

type LoadedFile = { kind: string; start: number; length: number; bytes: number[] };
type EmulatorWindow = Window & {
  __v06EmbeddedInputBridge?: boolean;
  Loader: new (
    url: string,
    rom: (bytes: Uint8Array, start: number) => void,
    error: () => void,
    fdd: (bytes: Uint8Array, start: number) => void,
  ) => unknown;
};

const rom = Buffer.from([0xc3, 0x00, 0x01, 0x42]);
const disk = Buffer.alloc(819200, 0xe5);

for (const scenario of [
  { name: "demo.rom", body: rom, kind: "rom", start: 0x100, length: rom.length, bytes: [...rom] },
  { name: "demo.fdd", body: disk, kind: "fdd", start: 0, length: disk.length, bytes: [0xe5, 0xe5, 0xe5, 0xe5] },
  { name: "rom.zip", body: archive("demo.rom", rom), kind: "rom", start: 0x100, length: rom.length, bytes: [...rom] },
  { name: "disk.zip", body: archive("demo.fdd", disk), kind: "fdd", start: 0, length: disk.length, bytes: [0xe5, 0xe5, 0xe5, 0xe5] },
]) {
  test(`emulator loads ${scenario.name}`, async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route(/^https?:\/\/[^/]+\/fixtures\/demo\.rom$/, (route) => route.fulfill({ body: rom }));
    const fixtureUrl = new RegExp(`^https?://[^/]+/fixtures/${scenario.name.replaceAll(".", "\\.")}$`);
    await page.route(fixtureUrl, (route) => route.fulfill({ body: scenario.body }));
    await page.goto("/emulator/?i:/fixtures/demo.rom");
    await page.waitForFunction(() => (window as EmulatorWindow).__v06EmbeddedInputBridge);
    const loaded = await page.evaluate((filename) => new Promise<LoadedFile>((resolve, reject) => {
      const complete = (kind: string) => (data: Uint8Array, start: number) => {
        resolve({ kind, start, length: data.length, bytes: [...data.slice(0, 4)] });
      };
      new (window as EmulatorWindow).Loader(`/fixtures/${filename}`, complete("rom"),
        () => reject(new Error("Emulator file load failed")), complete("fdd"));
    }), scenario.name);
    expect(loaded).toEqual({ kind: scenario.kind, start: scenario.start, length: scenario.length, bytes: scenario.bytes });
    expect(errors).toEqual([]);
  });
}
