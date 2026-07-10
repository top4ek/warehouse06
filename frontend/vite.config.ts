import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";
import type { Plugin } from "vite";

/** Bust browser caches of old hashed index/entry bundles after deploy. */
function injectBuildId(): Plugin {
  const buildId = process.env.VITE_BUILD_ID ?? "dev";
  const reloadScript = `(function(){try{var k="warehouse06-build",b="${buildId}",p=localStorage.getItem(k);if(p&&p!==b){var rk="warehouse06-reloaded-for";if(sessionStorage.getItem(rk)===b){localStorage.setItem(k,b);return}sessionStorage.setItem(rk,b);localStorage.setItem(k,b);location.reload();return}if(b)localStorage.setItem(k,b)}catch(e){}})();`;
  return {
    name: "inject-build-id",
    transformIndexHtml(html) {
      const tags =
        `    <meta name="warehouse06-build" content="${buildId}" />\n    <script>${reloadScript}</script>\n`;
      let out = html;
      if (out.includes("<script type=\"module\"")) {
        out = out.replace("<script type=\"module\"", `${tags}    <script type="module"`);
      } else {
        out = out.replace("</head>", `${tags}  </head>`);
      }
      return out.replace(
        /(<script type="module" crossorigin src="\/assets\/index-[^"]+\.js)(">)/,
        `$1?v=${buildId}$2`,
      );
    },
  };
}

const proxyTarget = process.env.VITE_PROXY_TARGET ?? "http://localhost:8080";

/** Storage / catalog file extensions served by the Go backend, not Vite. */
const storageAsset =
  "^/.+\\.(rom|fdd|zip|com|bin|r0m|png|jpe?g|gif|webp|svg|pdf|txt|html?)$";

export default defineConfig({
  plugins: [react(), injectBuildId()],
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
  },
  server: {
    host: true,
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": { target: proxyTarget, changeOrigin: true },
      "/assets": { target: proxyTarget, changeOrigin: true },
      [storageAsset]: { target: proxyTarget, changeOrigin: true },
    },
  },
  build: {
    emptyOutDir: false,
    rollupOptions: {
      output: {
        chunkFileNames: (chunkInfo) => {
          const id = chunkInfo.facadeModuleId ?? "";
          if (id.includes("/pages/Entry")) return "assets/entry.js";
          return "assets/[name]-[hash].js";
        },
        manualChunks(id) {
          if (id.includes("node_modules")) {
            if (id.includes("@tanstack/react-query")) return "query";
            if (id.includes("antd") || id.includes("@ant-design")) return "antd";
            if (id.includes("react-router")) return "router";
            if (id.includes("react-dom") || id.includes("/react/")) return "react";
          }
        },
      },
    },
  },
});
