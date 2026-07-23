import { defineConfig } from "vite";

const MAINNET_RPC = "http://169.155.55.44:26657";
const MAINNET_REST = "http://169.155.55.44:1317";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: 4173,
    proxy: {
      "/api/rpc": {
        target: MAINNET_RPC,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/rpc/, "") || "/",
      },
      "/api/rest": {
        target: MAINNET_REST,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/rest/, "") || "/",
      },
    },
  },
  build: {
    target: "es2022",
    sourcemap: false,
  },
});
