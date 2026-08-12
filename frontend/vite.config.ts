import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
		host: "127.0.0.1",
		port: 43177,
		strictPort: true,
    proxy: {
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
});
