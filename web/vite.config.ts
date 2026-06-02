import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

const backendTarget = process.env.VITE_PROXY_TARGET ?? process.env.VITE_API_BASE_URL ?? "http://localhost:18080";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5174,
    proxy: {
      "/readyz": backendTarget,
      "/healthz": backendTarget,
      "/api": backendTarget,
      "/admin/api": backendTarget,
    },
  },
});
