import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

const proxyTarget = process.env.VITE_PROXY_TARGET ?? "http://localhost:8080";

/** Storage / catalog file extensions served by the Go backend, not Vite. */
const storageAsset =
  "^/.+\\.(rom|fdd|zip|com|bin|r0m|png|jpe?g|gif|webp|svg|pdf|txt|html?)$";

export default defineConfig({
  plugins: [react()],
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
