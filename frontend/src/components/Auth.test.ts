import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";

describe("Auth", () => {
  afterEach(cleanup);

  it("has a main landmark region", () => {

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
});
