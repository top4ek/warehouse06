import { describe, expect, it } from "vitest";
import { canPlayWav, makeTapeWav } from "./tapeAudio";

async function ascii(blob: Blob, offset: number): Promise<string> {
  const bytes = new Uint8Array(await blob.arrayBuffer());
  return String.fromCharCode(...bytes.slice(offset, offset + 4));
}

describe("canPlayWav", () => {
  it("accepts tape-encodable binaries, case-insensitively", () => {
    for (const name of ["game.rom", "GAME.ROM", "loader.com", "x.bin", "demo.r0m", "a.R0M"]) {
      expect(canPlayWav(name)).toBe(true);
    }
  });

  it("rejects non-tape files (fdd/zip/images/m01/etc.)", () => {
    for (const name of ["disk.fdd", "pack.zip", "art.png", "mon.m01", "readme.txt", "noext"]) {
      expect(canPlayWav(name)).toBe(false);
    }
  });
});

describe("makeTapeWav", () => {
  const data = new Uint8Array(64).map((_, i) => i & 0xff);

  it("produces a valid WAV blob", async () => {
    const blob = makeTapeWav(data, "game.rom");
    expect(blob.type).toBe("audio/wav");
    expect(blob.size).toBeGreaterThan(44);
    expect(await ascii(blob, 0)).toBe("RIFF");
    expect(await ascii(blob, 8)).toBe("WAVE");
  });

  it("encodes .r0m (load address 0) differently from the 0x100 default", async () => {
    const rom = new Uint8Array(await makeTapeWav(data, "game.rom").arrayBuffer());
    const r0m = new Uint8Array(await makeTapeWav(data, "game.r0m").arrayBuffer());
    expect(r0m).not.toEqual(rom);
  });

  it("is deterministic for the same input", async () => {
    const a = new Uint8Array(await makeTapeWav(data, "game.rom").arrayBuffer());
    const b = new Uint8Array(await makeTapeWav(data, "game.rom").arrayBuffer());
    expect(a).toEqual(b);
  });
});
