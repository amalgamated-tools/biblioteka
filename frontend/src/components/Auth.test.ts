import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    loading: false,
    user: null,
    signIn: vi.fn(),
    signUp: vi.fn(),
  },
}));

vi.mock("../lib/api", () => ({
  getOidcEnabled: vi.fn().mockResolvedValue(false),
}));

vi.mock("lucide-svelte", () => ({ BookCheck: () => {} }));
vi.mock("./ui/AlertBanner.svelte", () => ({ default: () => {} }));

import Auth from "./Auth.svelte";

describe("Auth", () => {
  it("has a main landmark region", () => {
    render(Auth);

    const main = screen.getByRole("main");
    expect(main).toBeInTheDocument();
  });

  it("uses accessible placeholder contrast classes", () => {
    const { container } = render(Auth);

    const inputs = Array.from(
      container.querySelectorAll("input[placeholder]"),
    ) as HTMLInputElement[];

    expect(inputs.length).toBeGreaterThan(0);
    for (const input of inputs) {
      expect(input.className).toContain("placeholder:text-ink-400");
      expect(input.className).not.toContain("placeholder:text-ink-300");
    }
  });
});
