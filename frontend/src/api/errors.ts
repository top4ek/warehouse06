export function apiErrorMessage(status: number, body: string): string {
  if (body) {
    try {
      const parsed = JSON.parse(body) as { error?: string };
      if (parsed.error) return parsed.error;
    } catch {
      if (!body.startsWith("<")) return body;
    }
  }

  switch (status) {
    case 400:
      return "Invalid request";
    case 404:
      return "Not found";
    case 503:
      return "Catalog is updating, please wait…";
    default:
      return status >= 500 ? "Server error" : "Request failed";
  }
}
