import type { ThemeMode } from "../theme";

const FAVICONS: Record<ThemeMode, string> = {
  light: "/favicon.svg",
  dark: "/favicon-dark.svg",
};

function setIconLink(rel: string, href: string, marker: string) {
  let link = document.querySelector<HTMLLinkElement>(`link[data-warehouse06-${marker}]`);
  if (!link) {
    link = document.createElement("link");
    link.rel = rel;
    link.type = "image/svg+xml";
    link.setAttribute(`data-warehouse06-${marker}`, "");
    document.head.appendChild(link);
  }
  link.href = href;
}

export function applyFavicon(mode: ThemeMode) {
  const mark = mode === "dark" ? "/logo-mark-dark.svg" : "/logo-mark.svg";
  setIconLink("icon", FAVICONS[mode], "favicon");
  setIconLink("apple-touch-icon", mark, "apple-touch");
}
