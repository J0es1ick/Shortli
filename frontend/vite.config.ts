import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

export default defineConfig(({ mode }) => ({
  base: mode === "demo" ? "./" : "/",
  define: {
    "import.meta.env.VITE_DEMO_MODE": JSON.stringify(
      mode === "demo" ? "true" : "false",
    ),
    ...(mode === "demo"
      ? {
          "import.meta.env.VITE_API_BASE_URL": '""',
          "import.meta.env.VITE_PUBLIC_BASE_URL": '""',
        }
      : {}),
  },
  build: { outDir: mode === "demo" ? "dist-demo" : "dist" },
  resolve: {
    alias: {
      "@shortli/transport": fileURLToPath(
        new URL(
          mode === "demo" ? "./src/demo/api.ts" : "./src/lib/transport.ts",
          import.meta.url,
        ),
      ),
    },
  },
  plugins: [react()],
  server: {
    host: "127.0.0.1",
    port: 43177,
    strictPort: true,
    proxy:
      mode === "demo"
        ? undefined
        : {
            "/api": {
              target: "http://localhost:43178",
              changeOrigin: true,
              rewrite: (path) => path.replace(/^\/api/, "/api"),
            },
          },
  },
  preview: {
    host: "127.0.0.1",
    port: 43177,
    strictPort: true,
  },
}));
