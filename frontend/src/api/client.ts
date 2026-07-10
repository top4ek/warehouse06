import { z } from "zod";
import { apiErrorMessage } from "./errors";

type QueryValue = string | number | undefined | null;

export function buildUrl(path: string, params?: Record<string, QueryValue>) {
  const url = new URL(path, window.location.origin);
  Object.entries(params ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      url.searchParams.set(key, String(value));
    }
  });
  return `${url.pathname}${url.search}`;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

const maxRetryAttempts = 10;

export async function getJSON<T>(
  path: string,
  params: Record<string, QueryValue> | undefined,
  schema: z.ZodType<T>,
  signal?: AbortSignal,
): Promise<T> {
  for (let attempt = 0; attempt < maxRetryAttempts; attempt++) {
    const timeout = AbortSignal.timeout(30_000);
    const res = await fetch(buildUrl(path, params), {
      signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
    });
    if (res.status === 503 && attempt < maxRetryAttempts - 1) {
      const retryAfterSec = Number(res.headers.get("Retry-After")) || 2;
      await sleep(retryAfterSec * 1000);
      // The caller may have navigated away during the back-off; stop retrying.
      signal?.throwIfAborted();
      continue;
    }
    if (!res.ok) {
      const body = await res.text();
      throw new Error(apiErrorMessage(res.status, body));
    }
    const data: unknown = await res.json();
    const parsed = schema.safeParse(data);
    if (!parsed.success) {
      console.error(`unexpected response shape from ${path}:`, parsed.error);
      throw new Error("Unexpected response from server");
    }
    return parsed.data;
  }

  throw new Error(apiErrorMessage(503, ""));
}

export function storageUrl(path: string) {
  const clean = path.replace(/^\/+/, "");
  const encoded = clean.split("/").map(encodeURIComponent).join("/");
  return `/${encoded}`;
}

export function absoluteStorageUrl(path: string) {
  const relative = storageUrl(path);
  return `${window.location.origin}${relative}`;
}
