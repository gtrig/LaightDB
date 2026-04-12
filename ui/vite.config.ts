import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In Docker (laightdb-ui-dev), set VITE_API_PROXY=http://laightdb:8080 so /v1 reaches the API container.
const apiProxyTarget =
  process.env.VITE_API_PROXY ?? "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      "/v1": {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
});
