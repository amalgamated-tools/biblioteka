import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "account",
  },
}));

vi.mock("../lib/api", () => ({
  getConfigStatus: vi.fn().mockResolvedValue({
    oidc_configured: false,
    smtp_configured: false,
    is_admin: false,
  }),
  getOidcConfig: vi.fn(),
}));

// Mock all settings tab sub-components so the tests focus on Settings navigation
vi.mock("./settings/AccountTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/OidcTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/SmtpTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/UsersTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/PreferencesTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/APIKeysTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/KoboTab.svelte", () => ({ default: () => {} }));

vi.mock("lucide-svelte", () => ({
  Mail: () => {},
  Palette: () => {},
  Shield: () => {},
  Users: () => {},
  Send: () => {},
  KeyRound: () => {},
  BookOpen: () => {},
}));

import Settings from "./Settings.svelte";
import { routerStore } from "../stores/router.svelte";
import { getConfigStatus } from "../lib/api";

describe("Settings navigation", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the Settings heading", async () => {
    render(Settings);
    await tick();

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "Settings",
    );
  });

  it("renders user-facing navigation links", async () => {
    render(Settings);
    await tick();

    expect(screen.getByRole("link", { name: /Account/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /API Keys/i })).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Preferences/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Kobo Sync/i }),
    ).toBeInTheDocument();
  });

  it("does not render admin-only navigation links for non-admin users", async () => {
    vi.mocked(getConfigStatus).mockResolvedValue({
      oidc_configured: false,
      smtp_configured: false,
      is_admin: false,
    });
    render(Settings);
    await tick();
    await tick();

    expect(screen.queryByRole("link", { name: /OIDC \/ SSO/i })).toBeNull();
    expect(screen.queryByRole("link", { name: /Email \/ SMTP/i })).toBeNull();
    expect(screen.queryByRole("link", { name: /Users/i })).toBeNull();
  });

  it("renders admin-only navigation links for admin users", async () => {
    vi.mocked(getConfigStatus).mockResolvedValue({
      oidc_configured: false,
      smtp_configured: false,
      is_admin: true,
    });
    render(Settings);
    await tick();
    await tick();

    expect(
      screen.getByRole("link", { name: /OIDC \/ SSO/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Email \/ SMTP/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Users/i })).toBeInTheDocument();
  });

  it("sets aria-current='page' on the active tab link", async () => {
    vi.mocked(routerStore).subPath = "account";
    render(Settings);
    await tick();

    const accountLink = screen.getByRole("link", { name: /Account/i });
    expect(accountLink).toHaveAttribute("aria-current", "page");
  });

  it("does not set aria-current on inactive tab links", async () => {
    vi.mocked(routerStore).subPath = "account";
    render(Settings);
    await tick();

    const preferencesLink = screen.getByRole("link", { name: /Preferences/i });
    expect(preferencesLink).not.toHaveAttribute("aria-current");
  });

  it("sets aria-current='page' on the Preferences tab when subPath is 'preferences'", async () => {
    vi.mocked(routerStore).subPath = "preferences";
    render(Settings);
    await tick();

    const preferencesLink = screen.getByRole("link", { name: /Preferences/i });
    expect(preferencesLink).toHaveAttribute("aria-current", "page");
  });

  it("defaults to account tab when subPath is empty", async () => {
    vi.mocked(routerStore).subPath = "";
    render(Settings);
    await tick();

    const accountLink = screen.getByRole("link", { name: /Account/i });
    expect(accountLink).toHaveAttribute("aria-current", "page");
  });

  it("calls getConfigStatus on mount", async () => {
    render(Settings);
    await tick();
    await tick();

    expect(getConfigStatus).toHaveBeenCalled();
  });

  it("renders the nav landmark with accessible label", async () => {
    render(Settings);
    await tick();

    expect(
      screen.getByRole("navigation", { name: /Settings sections/i }),
    ).toBeInTheDocument();
  });
});
