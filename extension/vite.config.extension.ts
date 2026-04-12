import { defineConfig } from "vite";
import { resolve } from "path";
import { copyFileSync, mkdirSync } from "fs";

// Copy the manifest and static assets into dist/ after each build.
function copyManifestPlugin() {
  return {
    name: "copy-manifest",
    closeBundle() {
      mkdirSync("dist/popup", { recursive: true });
      mkdirSync("dist/background", { recursive: true });
      mkdirSync("dist/icons", { recursive: true });
      copyFileSync("manifest.json", "dist/manifest.json");
      copyFileSync("src/popup/popup.html", "dist/popup/popup.html");
      copyFileSync("src/popup/popup.css", "dist/popup/popup.css");
      // Copy the source SVG icon. For store submissions, generate PNG icons
      // from this SVG at 16×16, 32×32, 48×48, and 128×128 and place them in
      // extension/src/icons/ before running `make build-extension`.
      copyFileSync("src/icons/icon.svg", "dist/icons/icon.svg");
    },
  };
}

export default defineConfig({
  root: ".",
  plugins: [copyManifestPlugin()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        popup: resolve(__dirname, "src/popup/popup.ts"),
        service_worker: resolve(__dirname, "src/background/service_worker.ts"),
      },
      output: {
        entryFileNames: (chunk) => {
          if (chunk.name === "service_worker") return "background/service_worker.js";
          if (chunk.name === "popup") return "popup/popup.js";
          return "[name]/[name].js";
        },
        chunkFileNames: "shared/[name]-[hash].js",
        // Keep modules as ES modules — required for service workers in MV3.
        format: "es",
      },
    },
    target: "esnext",
    sourcemap: false,
    minify: false,
  },
});
