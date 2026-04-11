import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";

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
  getSignupEnabled: vi.fn().mockResolvedValue(true),
}));

vi.mock("lucide-svelte", () => ({ BookCheck: () => {} }));

import { getOidcEnabled, getSignupEnabled } from "../lib/api";

import Auth from "./Auth.svelte";

describe("Auth", () => {
  afterEach(() => {
    cleanup();
  });

  const renderAuth = async () => {
    render(Auth);
    await tick();
  };

  it("has a main landmark region", async () => {
    await renderAuth();

    const main = screen.getByRole("main");
    expect(main).toBeInTheDocument();
  });

  it("renders tab buttons with correct ARIA attributes", async () => {
    await renderAuth();

    const tabList = screen.getByRole("tablist");
    expect(tabList).toBeInTheDocument();

    const loginTab = screen.getByRole("tab", { name: "Login" });
    expect(loginTab).toHaveAttribute("aria-selected", "true");
    expect(loginTab).toHaveAttribute("aria-controls", "login-panel");

    const signupTab = screen.getByRole("tab", { name: "Sign Up" });
    expect(signupTab).toHaveAttribute("aria-selected", "false");
    expect(signupTab).toHaveAttribute("aria-controls", "signup-panel");
  });

  it("renders tab panels with correct ARIA attributes", async () => {
    await renderAuth();

    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    expect(panels).toHaveLength(2);

    const loginPanel = panels.find((p) => p.id === "login-panel");
    expect(loginPanel).toBeDefined();
    expect(loginPanel!).toHaveAttribute("aria-labelledby", "login-tab");

    const signupPanel = panels.find((p) => p.id === "signup-panel");
    expect(signupPanel).toBeDefined();
    expect(signupPanel!).toHaveAttribute("aria-labelledby", "signup-tab");
  });

  it("switches ARIA state when the Sign Up tab is clicked", async () => {
    await renderAuth();

    const user = userEvent.setup();
    const signupTab = screen.getByRole("tab", { name: "Sign Up" });
    await user.click(signupTab);
    await tick();

    expect(signupTab).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Login" })).toHaveAttribute(
      "aria-selected",
      "false",
    );

    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    const signupPanel = panels.find((p) => p.id === "signup-panel");
    expect(signupPanel).toBeDefined();
    expect(signupPanel!).not.toHaveAttribute("hidden");
  });

  it("hides the Sign Up tab and panel when signup is disabled", async () => {
    vi.mocked(getSignupEnabled).mockResolvedValueOnce(false);

    render(Auth);
    await vi.waitFor(() => {
      expect(screen.queryByRole("tab", { name: "Sign Up" })).toBeNull();
    });

    const loginTab = screen.getByRole("tab", { name: "Login" });
    expect(loginTab).toBeInTheDocument();

    // Signup panel should not be rendered when signup is disabled
    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    expect(panels).toHaveLength(1);
    expect(panels[0].id).toBe("login-panel");
  });

  it("keyboard navigation does not hide login panel when signup is disabled", async () => {
    vi.mocked(getSignupEnabled).mockResolvedValueOnce(false);

    const user = userEvent.setup();
    render(Auth);
    await vi.waitFor(() => {
      expect(screen.queryByRole("tab", { name: "Sign Up" })).toBeNull();
    });

    const loginTab = screen.getByRole("tab", { name: "Login" });
    loginTab.focus();

    // ArrowRight should not toggle away from login
    await user.keyboard("{ArrowRight}");
    await tick();

    const loginPanel = screen.getByRole("tabpanel", { hidden: true });
    expect(loginPanel.id).toBe("login-panel");
    expect(loginPanel).not.toHaveAttribute("hidden");
  });

  it("shows an error banner when OIDC check fails", async () => {
    vi.mocked(getOidcEnabled).mockRejectedValueOnce(new Error("network"));

    render(Auth);
    await vi.waitFor(() => {
      expect(
        screen.getByText("Unable to reach the server to load auth settings"),
      ).toBeInTheDocument();
    });
  });

  it("shows an error banner when signup check fails", async () => {
    vi.mocked(getSignupEnabled).mockRejectedValueOnce(new Error("network"));

    render(Auth);
    await vi.waitFor(() => {
      expect(
        screen.getByText("Unable to reach the server to load auth settings"),
      ).toBeInTheDocument();
    });
  });
});
