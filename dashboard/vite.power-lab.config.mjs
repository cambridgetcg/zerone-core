import { defineConfig } from "vite";

// This is intentionally independent from vite.config.ts: the ordinary local
// dashboard proxies /api to zerone-1, while the power lab must have no path to
// a chain endpoint. It serves checked-in fixtures on loopback only.
export default defineConfig({
  appType: "mpa",
  clearScreen: false,
  server: {
    host: "127.0.0.1",
    port: 4174,
    strictPort: true,
    cors: false,
    headers: {
      "Cache-Control": "no-store",
      "X-Robots-Tag": "noindex, nofollow, noarchive",
    },
  },
});
