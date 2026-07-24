import { TapeFormat } from "../vendor/bin2wav/tape.js";

// Formats that can be encoded as Vector-06C tape audio. Kept separate from the
// emulator's playable list (Entry.tsx): .fdd/.zip are emulator-only, and the
// tape encoder targets raw ROM-style binaries.
const WAV_EXT = [".rom", ".com", ".bin", ".r0m"];

export function canPlayWav(filename: string): boolean {
  const lc = filename.toLowerCase();
  return WAV_EXT.some((ext) => lc.endsWith(ext));
}

/**
 * Encode a binary as a Vector-06C tape WAV, mirroring caglrc.cc's player:
 * .r0m loads at address 0, everything else strips the 256-byte header.
 */
export function makeTapeWav(data: Uint8Array, filename: string): Blob {
  const org = filename.toLowerCase().endsWith("r0m") ? 0 : 256;
  const stream = TapeFormat("v06c-rom").format(data, org, filename).makewav();
  return new Blob([stream], { type: "audio/wav" });
}
