import { describe, expect, it, vi, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/svelte";

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "account",
    navigate: vi.fn(),
  },
}));

vi.mock("../lib/api", () => ({
  getConfigStatus: vi.fn().mockResolvedValue({
    oidc_configured: false,
    smtp_configured: false,
    is_admin: true,
  }),
  getOidcConfig: vi.fn(),
  getSmtpConfig: vi.fn().mockResolvedValue({
    host: "",
    port: "587",
    username: "",
    from: "",
    tls: "starttls",
    env_override: false,
    password_set: false,
  }),
  setSmtpConfig: vi.fn(),
  testSmtpConfig: vi.fn(),
}));

vi.mock("./settings/AccountTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/OidcTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/UsersTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/PreferencesTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/APIKeysTab.svelte", () => ({ default: () => {} }));

vi.mock("lucide-svelte", () => ({
  Mail: () => {},
  Palette: () => {},
  Shield: () => {},
  Users: () => {},
  Send: () => {},
  KeyRound: () => {},
}));

import Settings from "./Settings.svelte";
import { routerStore } from "../stores/router.svelte";

afterEach(() => {
  cleanup();
});

describe("Settings", () => {
  describe("aria-current on navigation tabs", () => {
    it("sets aria-current='page' on Account tab when it is the active tab", () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (routerStore as any).subPath = "account";
      render(Settings);

      expect(
        screen.getByRole("link", { name: /account/i }),
      ).toHaveAttribute("aria-current", "page");

      expect(
        screen.getByRole("link", { name: /api keys/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("link", { name: /preferences/i }),
      ).not.toHaveAttribute("aria-current");
    });

    it("sets aria-current='page' on Preferences tab when it is the active tab", () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (routerStore as any).subPath = "preferences";
      render(Settings);

      expect(
        screen.getByRole("link", { name: /account/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("link", { name: /preferences/i }),
      ).toHaveAttribute("aria-current", "page");
    });

    it("sets aria-current='page' on admin tabs when active", async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (routerStore as any).subPath = "oidc";
      render(Settings);

      // Admin tabs render after onMount resolves getConfigStatus
      await waitFor(() => {
        expect(
          screen.getByRole("link", { name: /oidc/i }),
        ).toHaveAttribute("aria-current", "page");
      });

      expect(
        screen.getByRole("link", { name: /account/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("link", { name: /users/i }),
      ).not.toHaveAttribute("aria-current");
    });
  });
});
