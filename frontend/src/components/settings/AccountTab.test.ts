import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "u1",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    },
    oidcLinkError: null,
  },
}));

vi.mock("../../lib/api", () => ({
  changePassword: vi.fn().mockResolvedValue(undefined),
  createOidcLinkNonce: vi.fn().mockResolvedValue("nonce-abc"),
}));

vi.mock("lucide-svelte", () => ({
  Lock: () => {},
  Mail: () => {},
  Link: () => {},
}));

import AccountTab from "./AccountTab.svelte";
import { authStore } from "../../stores/auth.svelte";
import { changePassword, createOidcLinkNonce } from "../../lib/api";

describe("AccountTab email display", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("displays the user email in a read-only input", () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    const emailInput = screen.getByLabelText("Email Address");
    expect(emailInput).toHaveValue("test@example.com");
    expect(emailInput).toBeDisabled();
  });
});

describe("AccountTab password change", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders all three password fields", () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    expect(screen.getByLabelText("Current Password")).toBeInTheDocument();
    expect(screen.getByLabelText("New Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm New Password")).toBeInTheDocument();
  });

  it("shows validation error when current password is empty on submit", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Current password is required",
    );
  });

  it("shows validation error when new password is empty on submit", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "New password is required",
    );
  });

  it("shows validation error when new password is shorter than 6 characters", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "abc" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "New password must be at least 6 characters",
    );
  });

  it("shows validation error when confirm password does not match", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "different" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Passwords do not match",
    );
  });

  it("calls changePassword with correct values on valid submission", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(changePassword).toHaveBeenCalledWith("old-pass", "newpassword");
  });

  it("shows success message after password change", async () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("Password updated successfully"),
    ).toBeInTheDocument();
  });

  it("shows error banner when changePassword rejects", async () => {
    vi.mocked(changePassword).mockRejectedValueOnce(
      new Error("Current password incorrect"),
    );
    render(AccountTab, { props: { oidcConfigured: false } });

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "wrong-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Current password incorrect",
    );
  });

  describe("success message auto-dismiss", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("auto-dismisses the success message after 3 seconds", async () => {
      render(AccountTab, { props: { oidcConfigured: false } });

      await fireEvent.input(screen.getByLabelText("Current Password"), {
        target: { value: "old-pass" },
      });
      await fireEvent.input(screen.getByLabelText("New Password"), {
        target: { value: "newpassword" },
      });
      await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
        target: { value: "newpassword" },
      });

      const form = document.querySelector("form")!;
      await fireEvent.submit(form);
      await tick();
      await tick();

      expect(
        screen.getByText("Password updated successfully"),
      ).toBeInTheDocument();

      await vi.advanceTimersByTimeAsync(3000);
      await tick();

      expect(screen.queryByText("Password updated successfully")).toBeNull();
    });
  });
});

describe("AccountTab SSO section", () => {
  beforeEach(() => {
    vi.mocked(authStore).user = {
      id: "u1",
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

  it("hides the SSO section when oidcConfigured is false", () => {
    render(AccountTab, { props: { oidcConfigured: false } });

    expect(
      screen.queryByRole("heading", { name: /Single Sign-On/i }),
    ).toBeNull();
  });

  it("shows the SSO section when oidcConfigured is true", () => {
    render(AccountTab, { props: { oidcConfigured: true } });

    expect(
      screen.getByRole("heading", { name: /Single Sign-On/i }),
    ).toBeInTheDocument();
  });

  it("shows 'Link SSO Account' button when not yet linked", () => {
    vi.mocked(authStore).user = {
      id: "u1",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    };
    render(AccountTab, { props: { oidcConfigured: true } });

    expect(
      screen.getByRole("button", { name: /Link SSO Account/i }),
    ).toBeInTheDocument();
  });

  it("shows 'SSO Connected' status when user is already linked", () => {
    vi.mocked(authStore).user = {
      id: "u1",
      email: "test@example.com",
      oidc_linked: true,
      is_admin: false,
    };
    render(AccountTab, { props: { oidcConfigured: true } });

    expect(screen.getByText("SSO Connected")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Link SSO Account/i }),
    ).toBeNull();
  });

  it("calls createOidcLinkNonce when the Link SSO Account button is clicked", async () => {
    vi.mocked(authStore).user = {
      id: "u1",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    };
    render(AccountTab, { props: { oidcConfigured: true } });

    await fireEvent.click(
      screen.getByRole("button", { name: /Link SSO Account/i }),
    );
    await tick();

    expect(createOidcLinkNonce).toHaveBeenCalled();
  });

  it("shows error banner when oidcLinkError is set on the store", () => {
    vi.mocked(authStore).user = {
      id: "u1",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    };
    vi.mocked(authStore).oidcLinkError = "SSO provider is unavailable";
    render(AccountTab, { props: { oidcConfigured: true } });

    expect(screen.getByRole("alert")).toHaveTextContent(
      "SSO provider is unavailable",
    );
  });
});
