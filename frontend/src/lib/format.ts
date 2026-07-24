const addedFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

export function formatAddedAt(iso?: string): string | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  return addedFormatter.format(date);
}

const byteUnits = ["B", "KB", "MB", "GB", "TB"];

export function formatBytes(bytes?: number): string | null {
  if (bytes == null || Number.isNaN(bytes) || bytes < 0) return null;
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < byteUnits.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const rounded = unit === 0 ? value : Math.round(value * 10) / 10;
  return `${rounded} ${byteUnits[unit]}`;
}
