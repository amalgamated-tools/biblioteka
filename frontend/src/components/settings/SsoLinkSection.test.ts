import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "u1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    },
    oidcLinkError: null,
  },
}));

vi.mock("../../lib/api", () => ({
  createOidcLinkNonce: vi.fn().mockResolvedValue("nonce-abc"),
}));

vi.mock("lucide-svelte", () => ({
  Link: () => {},
}));

import SsoLinkSection from "./SsoLinkSection.svelte";
import { authStore } from "../../stores/auth.svelte";
import { createOidcLinkNonce } from "../../lib/api";

describe("SsoLinkSection", () => {
  beforeEach(() => {
    vi.mocked(authStore).user = {
      id: "u1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    };
    vi.mocked(authStore).oidcLinkError = null;
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders nothing when oidcConfigured is false", () => {
    const { container } = render(SsoLinkSection, {
      props: { oidcConfigured: false },
    });

    expect(container.querySelector("div")).toBeNull();
  });

  it("renders the SSO section when oidcConfigured is true", () => {
    render(SsoLinkSection, { props: { oidcConfigured: true } });

    expect(
      screen.getByRole("heading", { name: /Single Sign-On/i }),
    ).toBeInTheDocument();
  });

  it("shows 'Link SSO Account' button when user is not linked", () => {
    vi.mocked(authStore).user = {
      id: "u1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    };

    render(SsoLinkSection, { props: { oidcConfigured: true } });

    expect(
      screen.getByRole("button", { name: /Link SSO Account/i }),
    ).toBeInTheDocument();
  });

  it("shows 'SSO Connected' status when user is already linked", () => {
    vi.mocked(authStore).user = {
      id: "u1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: true,
      is_admin: false,
    };

    render(SsoLinkSection, { props: { oidcConfigured: true } });

    expect(screen.getByText("SSO Connected")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Link SSO Account/i }),
    ).toBeNull();
  });

  it("calls createOidcLinkNonce when the Link SSO Account button is clicked", async () => {
    const originalLocation = window.location;
    try {
      Object.defineProperty(window, "location", {
        value: { href: "" },
        writable: true,
      });

      render(SsoLinkSection, { props: { oidcConfigured: true } });

      await fireEvent.click(
        screen.getByRole("button", { name: /Link SSO Account/i }),
      );
      await tick();

      expect(createOidcLinkNonce).toHaveBeenCalled();
    } finally {
      Object.defineProperty(window, "location", {
        value: originalLocation,
        writable: true,
      });
    }
  });

  it("redirects to the OIDC link URL after receiving a nonce", async () => {
    const originalLocation = window.location;
    try {
      Object.defineProperty(window, "location", {
        value: { href: "" },
        writable: true,
      });

      render(SsoLinkSection, { props: { oidcConfigured: true } });

      await fireEvent.click(
        screen.getByRole("button", { name: /Link SSO Account/i }),
      );
      await tick();
      await tick();

      expect(window.location.href).toBe("/api/auth/oidc/link?nonce=nonce-abc");
    } finally {
      Object.defineProperty(window, "location", {
        value: originalLocation,
        writable: true,
      });
    }
  });

  it("shows error banner when oidcLinkError is set on the store", () => {
    vi.mocked(authStore).oidcLinkError = "SSO provider is unavailable";

    render(SsoLinkSection, { props: { oidcConfigured: true } });

    expect(screen.getByRole("alert")).toHaveTextContent(
      "SSO provider is unavailable",
    );
  });

  it("sets oidcLinkError when createOidcLinkNonce rejects", async () => {
    vi.mocked(createOidcLinkNonce).mockRejectedValueOnce(
      new Error("SSO service down"),
    );

    render(SsoLinkSection, { props: { oidcConfigured: true } });

    await fireEvent.click(
      screen.getByRole("button", { name: /Link SSO Account/i }),
    );
    await tick();
    await tick();

    expect(vi.mocked(authStore).oidcLinkError).toBe("SSO service down");
  });
});
