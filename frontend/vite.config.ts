import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { writeFileSync } from "fs";
import { resolve } from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss(),
    {
      name: "restore-gitkeep",
      closeBundle() {
        writeFileSync(resolve(__dirname, "../internal/server/dist/.gitkeep"), "");
      },
    },
  ],
  build: {
    outDir: "../internal/server/dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
  },
});
