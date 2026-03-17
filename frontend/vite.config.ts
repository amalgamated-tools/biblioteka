import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { writeFileSync } from "fs";
import { resolve } from "path";

let gitkeepPath: string | null = null;

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss(),
    {
      name: "restore-gitkeep",
      configResolved(resolvedConfig) {
        if (resolvedConfig.command === "build") {
          gitkeepPath = resolve(resolvedConfig.build.outDir, ".gitkeep");
        } else {
          gitkeepPath = null;
        }
      },
      closeBundle() {
        if (!gitkeepPath) {
          return;
        }
        writeFileSync(gitkeepPath, "");
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
