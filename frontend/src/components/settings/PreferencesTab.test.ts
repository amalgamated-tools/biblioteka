import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/theme.svelte", () => ({
  themeStore: {
    preference: "auto",
    set: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({ Palette: () => {} }));

import PreferencesTab from "./PreferencesTab.svelte";
import { themeStore } from "../../stores/theme.svelte";

describe("PreferencesTab", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders all three theme buttons", () => {
    render(PreferencesTab);

    expect(screen.getByRole("button", { name: "light" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "dark" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "auto" })).toBeInTheDocument();
  });

  it("renders a fieldset with a Theme legend", () => {
    const { container } = render(PreferencesTab);

    const legend = container.querySelector("legend");
    expect(legend).toBeInTheDocument();
    expect(legend).toHaveTextContent("Theme");
  });

  it("sets aria-pressed='true' on the active theme button", () => {
    vi.mocked(themeStore).preference = "dark";
    render(PreferencesTab);

    expect(screen.getByRole("button", { name: "dark" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("sets aria-pressed='false' on inactive theme buttons", () => {
    vi.mocked(themeStore).preference = "dark";
    render(PreferencesTab);

    expect(screen.getByRole("button", { name: "light" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
    expect(screen.getByRole("button", { name: "auto" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("calls themeStore.set with the correct theme when a button is clicked", async () => {
    render(PreferencesTab);

    await fireEvent.click(screen.getByRole("button", { name: "light" }));
    await tick();

    expect(themeStore.set).toHaveBeenCalledWith("light");
  });

  it("calls themeStore.set with 'dark' when the dark button is clicked", async () => {
    render(PreferencesTab);

    await fireEvent.click(screen.getByRole("button", { name: "dark" }));
    await tick();

    expect(themeStore.set).toHaveBeenCalledWith("dark");
  });
});
