import { absoluteStorageUrl } from "../api";

/** URL for vector06js iframe (`?i:` + absolute ROM URL, as expected by main.js). */
export function emulatorFrameSrc(romUrl: string): string {
  const path = romUrl.replace(/^\/+/, "");
  const absolute = romUrl.startsWith("http") ? romUrl : absoluteStorageUrl(path);
  return `/emulator/?i:${absolute}`;
}
