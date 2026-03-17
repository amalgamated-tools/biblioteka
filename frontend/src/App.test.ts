import { describe, expect, it } from "vitest";
import appSource from "./App.svelte?raw";

describe("App", () => {
  it("includes a skip link before the sidebar and targets the main content", () => {
    expect(appSource).toContain('href="#main-content"');
    expect(appSource).toContain("Skip to main content");
    expect(appSource).toContain('<main id="main-content"');
    expect(appSource.indexOf("Skip to main content")).toBeLessThan(
      appSource.indexOf("<Sidebar"),
    );
  });
});
