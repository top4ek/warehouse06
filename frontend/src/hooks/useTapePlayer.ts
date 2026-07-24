import { useCallback, useEffect, useRef, useState } from "react";
import { makeTapeWav } from "../lib/tapeAudio";

/**
 * Single-instance tape-audio player. Fetches a binary, encodes it to a WAV tape
 * blob (bin2wav) and plays it through one shared <audio> element - only one file
 * plays at a time. `toggle` starts/stops a given row; `pendingKey` covers the
 * fetch+encode gap so the button can show a loading state.
 */
export function useTapePlayer() {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const urlRef = useRef<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  // Bumped on every start/stop so a superseded in-flight load is ignored.
  const requestRef = useRef(0);
  const [playingKey, setPlayingKey] = useState<string | null>(null);
  const [pendingKey, setPendingKey] = useState<string | null>(null);

  const releaseUrl = useCallback(() => {
    if (urlRef.current) {
      URL.revokeObjectURL(urlRef.current);
      urlRef.current = null;
    }
  }, []);

  const stop = useCallback(() => {
    requestRef.current += 1;
    abortRef.current?.abort();
    abortRef.current = null;
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    releaseUrl();
    setPendingKey(null);
    setPlayingKey(null);
  }, [releaseUrl]);

  const toggle = useCallback(
    async (key: string, url: string, filename: string) => {
      if (key === playingKey || key === pendingKey) {
        stop();
        return;
      }
      stop();
      const requestId = (requestRef.current += 1);
      const controller = new AbortController();
      abortRef.current = controller;
      setPendingKey(key);
      try {
        const res = await fetch(url, { signal: controller.signal });
        if (!res.ok) throw new Error(`tape fetch failed: ${res.status}`);
        const data = new Uint8Array(await res.arrayBuffer());
        if (requestRef.current !== requestId) return;

        const objectUrl = URL.createObjectURL(makeTapeWav(data, filename));
        urlRef.current = objectUrl;
        // Configure a fresh element fully before it escapes into the ref; the
        // previous one was already stopped by the stop() call above.
        const audio = new Audio(objectUrl);
        audio.volume = 0.25;
        audio.onended = () => stop();
        audioRef.current = audio;
        await audio.play();
        if (requestRef.current !== requestId) return;
        setPendingKey(null);
        setPlayingKey(key);
      } catch {
        // AbortError (superseded/stopped) and fetch/play failures land here.
        if (requestRef.current === requestId) {
          releaseUrl();
          setPendingKey(null);
          setPlayingKey(null);
        }
      }
    },
    [playingKey, pendingKey, stop, releaseUrl],
  );

  useEffect(() => () => stop(), [stop]);

  return { playingKey, pendingKey, toggle };
}
