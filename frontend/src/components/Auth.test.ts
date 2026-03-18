import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";

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
  it("has a functional skip link that targets the auth main landmark", async () => {
    render(Auth);

    const skipLink = screen.getByRole("link", {
      name: /skip to main content/i,
    });
    const main = screen.getByRole("main");

    expect(main).toBeInTheDocument();
    expect(main.id).toBe("auth-main");
    expect(main).toHaveAttribute("tabindex", "-1");

    skipLink.focus();
    expect(document.activeElement).toBe(skipLink);

    await fireEvent.click(skipLink);

    expect(document.activeElement).toBe(main);
  });
});
