import { useEffect, useRef } from "react";

type Options = {
  canLoadPrev: boolean;
  canLoadNext: boolean;
  /**
   * Changes whenever a loaded window has been applied. Recreating the
   * observers on each change makes observe() report the current intersection
   * state immediately, so a sentinel still inside the margin after a load
   * fires again without any user scrolling (IntersectionObserver alone only
   * fires on transitions).
   */
  resetKey: string | number;
};

export function useBidirectionalScroll(
  onLoadPrev: () => void,
  onLoadNext: () => void,
  { canLoadPrev, canLoadNext, resetKey }: Options,
) {
  const topRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Latest callbacks behind stable refs, so changing identities do not tear
  // down the observers mid-flight.
  const loadPrevRef = useRef(onLoadPrev);
  const loadNextRef = useRef(onLoadNext);
  useEffect(() => {
    loadPrevRef.current = onLoadPrev;
    loadNextRef.current = onLoadNext;
  });

  useEffect(() => {
    const node = topRef.current;
    if (!node || !canLoadPrev) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) loadPrevRef.current();
      },
      { rootMargin: "240px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [canLoadPrev, resetKey]);

  useEffect(() => {
    const node = bottomRef.current;
    if (!node || !canLoadNext) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) loadNextRef.current();
      },
      { rootMargin: "240px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [canLoadNext, resetKey]);

  return { topRef, bottomRef };
}
