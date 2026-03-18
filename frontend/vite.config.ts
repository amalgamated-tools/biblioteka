import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { writeFileSync } from "fs";
import { resolve } from "path";
import type { Plugin } from "vite";

function restoreGitkeep(): Plugin {
  let gitkeepPath: string | undefined;
  return {
    name: "restore-gitkeep",
    apply: "build",
    configResolved(config) {
      gitkeepPath = resolve(config.build.outDir, ".gitkeep");
    },
    closeBundle() {
      if (!gitkeepPath) return;
      writeFileSync(gitkeepPath, "");
    },
  };
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte(), tailwindcss(), restoreGitkeep()],
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
  resolve: {
    conditions: ["browser"],
  },
});
