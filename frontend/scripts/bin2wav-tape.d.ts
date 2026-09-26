/**
 * Type surface for the vendored bin2wav tape.js used by the tape-audio player.
 * Only the pieces lib/tapeAudio.ts relies on are declared.
 */
export interface TapeStream {
  /** Build the WAV byte stream (ArrayBuffer) for the encoded tape. */
  makewav(): ArrayBuffer;
}

export interface TapeEncoder {
  /**
   * Encode `data` for the given load address (`org`) and tape name.
   * Returns the encoder for chaining into makewav().
   */
  format(data: Uint8Array, org: number, name: string): TapeStream;
}

/** Create a tape encoder for a bin2wav format id (e.g. "v06c-rom"). */
export function TapeFormat(
  fmt: string,
  forfile?: boolean,
  konst?: number,
  leader?: number,
  sampleRate?: number,
): TapeEncoder;
