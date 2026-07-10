const youtubeIDPattern = /^[a-zA-Z0-9_-]{11}$/;

export function parseYouTubeId(raw: string | undefined | null): string {
  const id = (raw ?? "").trim();
  return youtubeIDPattern.test(id) ? id : "";
}
